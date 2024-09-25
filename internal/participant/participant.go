package participant

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/signing"
	"github.com/bnb-chain/tss-lib/v2/tss"
)

type Participant struct {
	ID          *tss.PartyID
	OutCh       chan tss.Message
	EndKeyGenCh chan *keygen.LocalPartySaveData
	EndSignCh   chan *common.SignatureData
	InCh        chan tss.Message // Channel use as the communication medium between participants
	ErrCh       chan error
	Party       tss.Party
	Done        chan bool
}

func (p *Participant) handleMsg(participants []*Participant) {
	for msg := range p.OutCh {
		fmt.Printf("handling message %v\n", msg)
		to := msg.GetTo()
		if to == nil {
			// Broadcast message
			for _, otherParticipant := range participants {
				// send to all but self
				if otherParticipant != p {
					otherParticipant.InCh <- msg
				}
			}
		} else {
			// Direct message
			for _, toID := range to {
				target := findParticipantByID(participants, toID)
				if target != nil {
					target.InCh <- msg
				}
			}
		}
	}
}

func (p *Participant) listenForIncomingMessages() {
	for msg := range p.InCh {
		fmt.Printf("received message %v from %s, to %s, by %s\n", msg, msg.GetFrom(), msg.GetTo(), p.ID)
		data, msgRouting, err := msg.WireBytes()
		if err != nil {
			fmt.Printf("failed to parse wire message: %s\n", err)
			p.ErrCh <- fmt.Errorf("failed to parse wire message: %s", err)
			continue
		}
		if _, err := p.Party.UpdateFromBytes(data, msgRouting.From, msgRouting.IsBroadcast); err != nil {
			fmt.Printf("failed to update from bytes: %s\n", err)
			p.ErrCh <- fmt.Errorf("failed to update from bytes: %s", err)
		}
	}
}

func findParticipantByID(participants []*Participant, id *tss.PartyID) *Participant {
	for _, p := range participants {
		if p.ID.KeyInt().Cmp(id.KeyInt()) == 0 {
			return p
		}
	}
	return nil
}

func Generate(n int, threshold int) error {
	if n < 1 {
		return fmt.Errorf("n must be at least 2")
	}

	fmt.Printf("generating key pair with %d shares and threshold %d\n", n, threshold)

	parties := getParticipantPartyIDs(n)
	ctx := tss.NewPeerContext(parties)
	curve := tss.S256()
	participants := make([]*Participant, n)

	for idx, id := range parties {
		outCh := make(chan tss.Message, 2000)
		endCh := make(chan *keygen.LocalPartySaveData, 1)
		inCh := make(chan tss.Message, 2000)
		errCh := make(chan error, 1)
		doneCh := make(chan bool, 1)
		preParams, err := keygen.GeneratePreParams(1 * time.Minute)
		if err != nil {
			return fmt.Errorf("failed to generate pre-parameters: %w", err)
		}

		params := tss.NewParameters(curve, ctx, id, n, threshold)
		party := keygen.NewLocalParty(params, outCh, endCh, *preParams)

		participants[idx] = &Participant{
			ID:          id,
			OutCh:       outCh,
			EndKeyGenCh: endCh,
			InCh:        inCh,
			ErrCh:       errCh,
			Party:       party,
			Done:        doneCh,
		}
	}

	for _, p := range participants {
		// Start message handling for each participant
		go p.handleMsg(participants)
		// Listen for incoming messages
		go p.listenForIncomingMessages()
		// Start the party protocol
		go func(p *Participant) {
			if err := p.Party.Start(); err != nil {
				p.ErrCh <- err
			}
		}(p)
		go func(p *Participant) {
			for err := range p.ErrCh {
				fmt.Printf("Error: %s", err)
			}
		}(p)
		go func(p *Participant) {
			for save := range p.EndKeyGenCh {
				fmt.Println("Key generation protocol complete!")

				out, err := json.Marshal(*save)
				if err != nil {
					fmt.Printf("failed serializing output: %v", err)
				}

				fmt.Printf("Save data: %v\n", out)
				fmt.Printf("public key: %v\n", save.P)

				// dump to a file
				os.WriteFile(fmt.Sprintf("keygen-%s.json", p.ID.KeyInt().String()), out, 0644)

				// Indicate party completion
				p.Done <- true
				close(p.OutCh)
				close(p.InCh)
				close(p.ErrCh)
				close(p.Done)
				close(p.EndKeyGenCh)
				fmt.Println("Closed channels")
			}
		}(p)
	}

	// Wait for all participants to finish
	for _, p := range participants {
		<-p.Done
	}
	fmt.Println("All participants finished")
	return nil
}

func Load(n int, threshold int) ([]*keygen.LocalPartySaveData, error) {
	// list all file starting with keygen-
	files, err := listFilesWithPrefix(".", "keygen-")
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	keys := make([]*keygen.LocalPartySaveData, n)

	for idx, file := range files {
		// unmarshal the file
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %w", err)
		}

		var saveData keygen.LocalPartySaveData
		if err := json.Unmarshal(data, &saveData); err != nil {
			return nil, fmt.Errorf("failed to unmarshal save data: %w", err)
		}

		keys[idx] = &saveData
	}

	return keys, nil
}

func Sign(n int, threshold int, msg string) error {
	fmt.Printf("signing message %s\n", msg)

	keys, err := Load(n, threshold)
	if err != nil {
		return fmt.Errorf("failed to load keys: %w", err)
	}

	if len(keys) != n {
		return fmt.Errorf("expected %d keys, got %d", n, len(keys))
	}

	parties := getParticipantPartyIDs(n)
	ctx := tss.NewPeerContext(parties)
	curve := tss.S256()
	participants := make([]*Participant, n)

	payload := prepareMessage(msg)

	for idx, id := range parties {
		outCh := make(chan tss.Message, 2000)
		endCh := make(chan *common.SignatureData, 1)
		inCh := make(chan tss.Message, 2000)
		errCh := make(chan error, 1)
		doneCh := make(chan bool, 1)

		params := tss.NewParameters(curve, ctx, id, n, threshold)
		party := signing.NewLocalParty(
			payload,
			params,
			*keys[idx],
			outCh,
			endCh,
		)

		participants[idx] = &Participant{
			ID:        id,
			OutCh:     outCh,
			EndSignCh: endCh,
			InCh:      inCh,
			ErrCh:     errCh,
			Party:     party,
			Done:      doneCh,
		}
	}

	for _, p := range participants {
		// Start message handling for each participant
		go p.handleMsg(participants)
		// Listen for incoming messages
		go p.listenForIncomingMessages()
		// Start the party protocol
		go func(p *Participant) {
			if err := p.Party.Start(); err != nil {
				p.ErrCh <- err
			}
		}(p)
		go func(p *Participant) {
			for err := range p.ErrCh {
				fmt.Printf("Error: %s", err)
			}
		}(p)
		go func(p *Participant) {
			for save := range p.EndSignCh {
				fmt.Println("signing protocol complete!")

				out, err := json.Marshal(*save)
				if err != nil {
					fmt.Printf("failed serializing output: %v", err)
				}

				os.WriteFile(fmt.Sprintf("sig-%s.json", p.ID.KeyInt().String()), out, 0644)

				// Indicate party completion
				p.Done <- true
				close(p.OutCh)
				close(p.InCh)
				close(p.ErrCh)
				close(p.Done)
				close(p.EndSignCh)
				fmt.Println("Closed channels")
			}
		}(p)
	}

	// Wait for all participants to finish
	for _, p := range participants {
		<-p.Done
	}
	fmt.Println("All participants finished")
	return nil
}

func listFilesWithPrefix(dir, prefix string) ([]string, error) {
	var filesWithPrefix []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if strings.HasPrefix(filepath.Base(path), prefix) {
			filesWithPrefix = append(filesWithPrefix, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return filesWithPrefix, nil
}

func prepareMessage(msg string) *big.Int {
	byteArray := []byte(msg)
	hexString := hex.EncodeToString(byteArray)

	msg_bytes, err := hex.DecodeString(hexString)
	if err != nil {
		fmt.Println("Error:", err)
	}

	return new(big.Int).SetBytes(msg_bytes)
}

// source: https://github.com/flock-org/flock/blob/6b723c035599c0013f1a65d8ce1d18d53f775b1c/applications/internal/signing/signing-util.go#L81

// Return list of participant IDs
func getParticipantPartyIDs(numParties int) tss.SortedPartyIDs {
	var partyIds tss.UnSortedPartyIDs
	for i := 1; i <= numParties; i++ {
		partyIds = append(partyIds, tss.NewPartyID(strconv.Itoa(i), "", big.NewInt(int64(i))))
	}
	return tss.SortPartyIDs(partyIds)
}

package participant

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"github.com/Thykof/tss-lib-cli/internal/utils"
	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/signing"
	"github.com/bnb-chain/tss-lib/v2/tss"
)

// LoadKeys loads the keys from the files, return at least one key, or an error
func LoadKeys() ([]*keygen.LocalPartySaveData, error) {
	// list all file starting with keygen-
	files, err := utils.ListFilesWithPrefix(".", "keygen-")
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no key files found")
	}

	keys := make([]*keygen.LocalPartySaveData, len(files))

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

	keys, err := LoadKeys()
	if err != nil {
		return fmt.Errorf("failed to load keys: %w", err)
	}

	numberOfSigners := threshold + 1

	if len(keys) < numberOfSigners {
		return fmt.Errorf("expected at least %d keys, got %d", numberOfSigners, len(keys))
	}

	if threshold >= n {
		return fmt.Errorf("threshold must be less than n")
	}

	parties := utils.GetParticipantPartyIDs(numberOfSigners)
	ctx := tss.NewPeerContext(parties)
	curve := tss.S256()
	participants := make([]*Participant, numberOfSigners)

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

	for i := 0; i < numberOfSigners; i++ {
		p := participants[i]
		fmt.Printf("Starting participant %s\n", p.ID.KeyInt().String())
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
				panic(err)
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

func HashMessage(msg string) []byte {
	byteArray := []byte(msg)
	hexString := hex.EncodeToString(byteArray)

	msgBytes, err := hex.DecodeString(hexString)
	if err != nil {
		fmt.Println("Error:", err)
	}

	return msgBytes
}

func prepareMessage(msg string) *big.Int {
	return new(big.Int).SetBytes(HashMessage(msg))
}

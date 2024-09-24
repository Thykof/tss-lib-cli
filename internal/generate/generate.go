package generate

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/v2/tss"
)

type Participant struct {
	ID     *tss.PartyID
	OutCh  chan tss.Message
	EndCh  chan *keygen.LocalPartySaveData
	InCh   chan tss.Message // Channel use as the communication medium between participants
	ErrCh  chan error
	Party  tss.Party
	Done   chan bool
}

func Generate(n int, threshold int) error {
	if n < 1 {
		return fmt.Errorf("n must be at least 2")
	}

	fmt.Printf("generating key pair with %d shares and threshold %d\n", n, threshold)

	parties := tss.GenerateTestPartyIDs(n)
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
			ID:    id,
			OutCh: outCh,
			EndCh: endCh,
			InCh:  inCh,
			ErrCh: errCh,
			Party: party,
			Done:  doneCh,
		}
	}

	for _, p := range participants {
		// Start message handling for each participant
		go handleMsg(p, participants)
		// Listen for incoming messages
		go listenForIncomingMessages(p)
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
			for save := range p.EndCh {
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
				close(p.EndCh)
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

func handleMsg(participant *Participant, participants []*Participant) {
	for msg := range participant.OutCh {
		fmt.Printf("handling message %v\n", msg)
		to := msg.GetTo()
		if to == nil {
			// Broadcast message
			for _, p := range participants {
				// send to all but self
				if p != participant {
					p.InCh <- msg
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

func listenForIncomingMessages(participant *Participant) {
	for msg := range participant.InCh {
		fmt.Printf("received message %v from %s, to %s, by %s\n", msg, msg.GetFrom(), msg.GetTo(), participant.ID)
		data, msgRouting, err := msg.WireBytes()
		if err != nil {
			fmt.Printf("failed to parse wire message: %s\n", err)
			participant.ErrCh <- fmt.Errorf("failed to parse wire message: %s", err)
			continue
		}
		if _, err := participant.Party.UpdateFromBytes(data, msgRouting.From, msgRouting.IsBroadcast); err != nil {
			fmt.Printf("failed to update from bytes: %s\n", err)
			participant.ErrCh <- fmt.Errorf("failed to update from bytes: %s", err)
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

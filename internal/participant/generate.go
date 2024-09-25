package participant

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Thykof/tss-lib-cli/internal/utils"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/v2/tss"
)

func Generate(n int, threshold int) error {
	if n < 1 {
		return fmt.Errorf("n must be at least 2")
	}

	fmt.Printf("generating key pair with %d shares and threshold %d\n", n, threshold)

	parties := utils.GetParticipantPartyIDs(n)
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

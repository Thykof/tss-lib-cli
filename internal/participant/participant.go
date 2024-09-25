package participant

import (
	"fmt"

	"github.com/bnb-chain/tss-lib/v2/common"
	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
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

package utils

import (
	"math/big"
	"strconv"

	"github.com/bnb-chain/tss-lib/v2/tss"
)

// source: https://github.com/flock-org/flock/blob/6b723c035599c0013f1a65d8ce1d18d53f775b1c/applications/internal/signing/signing-util.go#L81

// Return list of participant IDs
func GetParticipantPartyIDs(numParties int) tss.SortedPartyIDs {
	var partyIds tss.UnSortedPartyIDs
	for i := 1; i <= numParties; i++ {
		partyIds = append(partyIds, tss.NewPartyID(strconv.Itoa(i), "", big.NewInt(int64(i))))
	}
	return tss.SortPartyIDs(partyIds)
}

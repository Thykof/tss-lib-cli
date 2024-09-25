package verifier

import (
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"github.com/Thykof/tss-lib-cli/internal/participant"
	"github.com/Thykof/tss-lib-cli/internal/utils"
	"github.com/bnb-chain/tss-lib/v2/common"
)

func LoadSig(numberOfSigners int) ([]*common.SignatureData, [][]byte, error) {
	// list all file starting with keygen-
	files, err := utils.ListFilesWithPrefix(".", "sig-")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list files: %w", err)
	}

	if len(files) != numberOfSigners {
		return nil, nil, fmt.Errorf("expected %d signatures, got %d", numberOfSigners, len(files))
	}

	signatures := make([]*common.SignatureData, numberOfSigners)
	contents := make([][]byte, numberOfSigners)

	for idx, file := range files {
		// unmarshal the file
		data, err := os.ReadFile(file)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read file: %w", err)
		}

		var sig common.SignatureData
		if err := json.Unmarshal(data, &sig); err != nil {
			return nil, nil, fmt.Errorf("failed to unmarshal save data: %w", err)
		}

		signatures[idx] = &sig
		contents[idx] = data
	}

	return signatures, contents, nil
}

func Verify(n int, threshold int, msg string) (bool, error) {
	numberOfSigners := threshold + 1

	fmt.Printf("verifying message %s\n", msg)
	signatures, contents, err := LoadSig(numberOfSigners)
	if err != nil {
		return false, fmt.Errorf("failed to load signatures: %w", err)
	}

	// Assert the number of files is equal to the number of participants
	if len(signatures) != numberOfSigners {
		return false, fmt.Errorf("expected %d signatures, got %d", numberOfSigners, len(signatures))
	}

	// Assert the file content are all the same
	for i := 1; i < numberOfSigners; i++ {
		if string(contents[i]) != string(contents[0]) {
			return false, fmt.Errorf("file content mismatch")
		}
	}

	keys, err := participant.LoadKeys(n)
	if err != nil {
		return false, fmt.Errorf("failed to load keys: %w", err)
	}

	publicKey := keys[0].ECDSAPub.ToECDSAPubKey()

	hash := participant.HashMessage(msg)

	r := new(big.Int).SetBytes(signatures[0].R)
	s := new(big.Int).SetBytes(signatures[0].S)

	return ecdsa.Verify(publicKey, hash, r, s), nil
}

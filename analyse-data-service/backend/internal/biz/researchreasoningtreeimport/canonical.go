package researchreasoningtreeimport

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

func CanonicalHash(publication Publication) (string, error) {
	payload, err := json.Marshal(publication)
	if err != nil {
		return "", fmt.Errorf("encode Research Reason Tree publication: %w", err)
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:]), nil
}

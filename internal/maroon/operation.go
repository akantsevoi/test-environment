package maroon

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/akantsevoi/test-environment/pkg/logger"
)

func (o *Operation) Hash() string {
	hash, _ := o.HashBin()

	return hash
}

func (o *Operation) HashBin() (string, []byte) {
	message, err := json.Marshal(o)
	if err != nil {
		logger.Fatalf(logger.Application, "Failed to marshal operation: %v", err)
	}

	return fmt.Sprintf("%x", sha256.Sum256(message)), message
}

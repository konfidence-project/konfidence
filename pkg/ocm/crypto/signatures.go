package crypto

import (
	"fmt"
	"strings"
)

// These constants are used to identify the signatures used across Konfidence.
const (
	// VectorAssemblySignature is the signature that is applied during the vector assembly process.
	VectorAssemblySignature = "konfidence.cloud.signature.vector.assembly"
	// ArtifactSignature is the signature that is applied during the artifact upload process.
	ArtifactSignature = "konfidence.cloud.signature.artifact.upload"
)

// signaturePreFlightSanityCheck validates signature names: no empty slice, no empty strings, no duplicates.
func signaturePreFlightSanityCheck(sigs []string) error {
	if len(sigs) == 0 {
		return fmt.Errorf("at least one target signature name must be provided")
	}
	seen := make(map[string]struct{}, len(sigs))
	for _, sig := range sigs {
		if strings.TrimSpace(sig) == "" {
			return fmt.Errorf("signature names cannot be empty or whitespace")
		}
		if _, exists := seen[sig]; exists {
			return fmt.Errorf("duplicate signature name detected: %q", sig)
		}
		seen[sig] = struct{}{}
	}
	return nil
}

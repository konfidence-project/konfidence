package controller

import (
	"fmt"
	"hash/fnv"
	"math/big"
	"strings"

	pkgsanitize "github.com/konfidence-project/konfidence/pkg/sanitize"
)

func ConstructArtifactDeploymentName(artifactName, artifactVersion string, uid *string) (string, error) {
	trimmedArtifactName := strings.TrimSpace(artifactName)
	trimmedArtifactVersion := strings.TrimSpace(artifactVersion)

	if len(trimmedArtifactName) == 0 || len(trimmedArtifactVersion) == 0 {
		return "", fmt.Errorf("artifact name or version is empty")
	}

	h := fnv.New128a()
	_, err := h.Write([]byte(trimmedArtifactName))
	if err != nil {
		return "", fmt.Errorf("unable to compute digest: %w", err)
	}

	_, err = h.Write([]byte(trimmedArtifactVersion))
	if err != nil {
		return "", fmt.Errorf("unable to compute digest: %w", err)
	}

	if uid != nil {
		// makes the name unique to this vector deployment -> no reuse
		_, err := h.Write([]byte(*uid))
		if err != nil {
			return "", fmt.Errorf("unable to compute digest: %w", err)
		}
	}

	hashBytes := h.Sum(nil)
	hash := new(big.Int).SetBytes(hashBytes).Text(36)
	versionWithHash := trimmedArtifactVersion + "-" + hash
	remainingSize := MaxLabelSize - len(versionWithHash)

	// use only hash value as name if version with hash is already too long
	if remainingSize < 0 {
		return pkgsanitize.DNSLabelName(hash), nil
	}

	// extract last part of artifact name
	componentName := trimmedArtifactName
	idx := strings.LastIndex(componentName, "/")
	if idx > -1 && idx != (len(componentName)-1) {
		componentName = componentName[idx+1:]
	}

	// truncate if necessary
	finalComponentName := componentName
	if len(componentName) > remainingSize-1 {
		finalComponentName = componentName[:remainingSize-1]
	}

	return pkgsanitize.DNSLabelName(finalComponentName + "-" + versionWithHash), nil
}

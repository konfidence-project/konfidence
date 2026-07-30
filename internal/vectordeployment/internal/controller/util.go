package controller

import (
	"fmt"
	"strconv"
	"strings"

	pkghash "github.com/konfidence-project/konfidence/pkg/hash"
	pkgsanitize "github.com/konfidence-project/konfidence/pkg/sanitize"
)

func ConstructArtifactDeploymentName(artifactName, artifactVersion string, uid *string, collisionCount int32) (name, hash string, err error) {
	trimmedArtifactName := strings.TrimSpace(artifactName)
	trimmedArtifactVersion := strings.TrimSpace(artifactVersion)

	if len(trimmedArtifactName) == 0 || len(trimmedArtifactVersion) == 0 {
		return "", "", fmt.Errorf("artifact name or version is empty")
	}

	// Hash over name + version (+ uid) as a single stream. When a uid is
	// supplied the name becomes unique to this vector deployment -> no reuse;
	// its absence yields a stable, reusable name across deployments.
	content := trimmedArtifactName + trimmedArtifactVersion
	if uid != nil {
		content += *uid
	}
	// Salt the hash on collision recovery. Only append when > 0.
	if collisionCount > 0 {
		content += strconv.FormatInt(int64(collisionCount), 10)
	}

	hash = pkghash.Fnv(content, 10)
	versionWithHash := trimmedArtifactVersion + "-" + hash
	remainingSize := MaxLabelSize - len(versionWithHash)

	// use only hash value as name if version with hash is already too long
	if remainingSize < 0 {
		return pkgsanitize.DNSLabelName(hash), hash, nil
	}
	// use version with hash if it matches the max output length
	if remainingSize == 0 {
		return pkgsanitize.DNSLabelName(versionWithHash), hash, nil
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

	return pkgsanitize.DNSLabelName(finalComponentName + "-" + versionWithHash), hash, nil
}

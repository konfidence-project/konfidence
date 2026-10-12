package artifactdeployment_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestArtifactDeployment(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ArtifactDeployment Domain Suite")
}

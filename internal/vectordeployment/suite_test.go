package vectordeployment_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestVectorDeployment(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "VectorDeployment Domain Suite")
}

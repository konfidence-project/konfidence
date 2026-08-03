package v1alpha1

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestV1Alpha1(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "V1Alpha1 Suite")
}

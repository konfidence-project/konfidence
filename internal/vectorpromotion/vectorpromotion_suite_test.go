package vectorpromotion

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestVectorPromotion(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "VectorPromotion Domain Suite")
}

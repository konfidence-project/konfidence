package promotiongate_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPromotionGate(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PromotionGate Suite")
}

package funcopts_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestFuncopts(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Funcopts Suite")
}

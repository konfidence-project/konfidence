package compref

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCompref(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Compref Suite")
}

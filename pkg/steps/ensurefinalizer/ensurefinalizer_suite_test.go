package ensurefinalizer_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEnsurefinalizer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ensurefinalizer Suite")
}

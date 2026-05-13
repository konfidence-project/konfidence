package ocm_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPkgOcm(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ocm pkg Suite")
}

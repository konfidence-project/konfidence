package content_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPkgOcmContent(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "pkg/ocm/content Suite")
}

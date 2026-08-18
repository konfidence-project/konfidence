package pretty

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPrettyFormat(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Format Pretty Suite")
}

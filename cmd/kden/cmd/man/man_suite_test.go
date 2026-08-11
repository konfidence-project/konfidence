package man_test

import (
	"testing"

	"github.com/adrg/xdg"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCmd(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Man Cmd Suite")
}

// The kden config file lives in the XDG config dir. Point it at a per-suite
// temp dir so parallel test packages never race on creating the real one.
var _ = BeforeSuite(func() {
	GinkgoT().Setenv("XDG_CONFIG_HOME", GinkgoT().TempDir())
	xdg.Reload()
})

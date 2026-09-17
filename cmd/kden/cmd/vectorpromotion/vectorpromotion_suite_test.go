package vectorpromotion_test

import (
	"testing"

	"github.com/adrg/xdg"
	cfg "github.com/konfidence-project/konfidence/internal/kden/config"
	"github.com/konfidence-project/konfidence/internal/kden/testutil"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCmd(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Vector Promotion Cmd Suite")
}

const (
	testVectorPromotionID       = "test-vector-promotion-id"
	flagVectorPromotionConfigID = "--vectorPromotionConfigId"
)

// The kden config file lives in the XDG config dir. Point it at a per-suite
// temp dir so parallel test packages never race on creating the real one.
var _ = BeforeSuite(func() {
	GinkgoT().Setenv("XDG_CONFIG_HOME", GinkgoT().TempDir())
	xdg.Reload()
	cfg.Config.Output = testutil.JSONOutputFormat
})

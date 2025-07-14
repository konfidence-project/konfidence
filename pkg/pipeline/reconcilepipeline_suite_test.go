package pipeline_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestReconcilePipeline(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ReconcilePipeline Suite")
}

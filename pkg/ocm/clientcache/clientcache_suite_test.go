package clientcache_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestClientCache(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ClientCache Suite")
}

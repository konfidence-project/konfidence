package lrucache_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestLRUCache(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "LRUCache Suite")
}

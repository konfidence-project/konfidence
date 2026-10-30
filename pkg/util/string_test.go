package util_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/konfidence-project/konfidence/pkg/util"
)

var _ = Describe("OptionalString", func() {
	It("returns nil for an empty string", func() {
		Expect(util.OptionalString("")).To(BeNil())
	})

	It("returns a pointer to a non-empty string", func() {
		p := util.OptionalString("hello")
		Expect(p).NotTo(BeNil())
		Expect(*p).To(Equal("hello"))
	})
})

package repository

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/go-logr/logr"
	"ocm.software/open-component-model/bindings/go/credentials"
	"ocm.software/open-component-model/bindings/go/runtime"
)

// staticResolver is a test-only credentials.Resolver that always returns a fixed value.
type staticResolver struct{}

func (staticResolver) Resolve(_ context.Context, _ runtime.Identity) (runtime.Typed, error) {
	return nil, credentials.ErrNotFound
}

var _ = Describe("OciClientBuilder", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	It("builds successfully with no configuration (anonymous access)", func() {
		client, err := NewOciClientBuilder().Build(ctx)
		Expect(err).NotTo(HaveOccurred())
		ociClient, ok := client.(OciClient)
		Expect(ok).To(BeTrue())
		_, isNoop := ociClient.resolver.(NoopCredentialResolver)
		Expect(isNoop).To(BeTrue(), "nil resolver should fall back to NoopCredentialResolver")
	})

	It("uses the provided resolver", func() {
		r := staticResolver{}
		client, err := NewOciClientBuilder().WithResolver(r).Build(ctx)
		Expect(err).NotTo(HaveOccurred())
		ociClient, ok := client.(OciClient)
		Expect(ok).To(BeTrue())
		_, isNoop := ociClient.resolver.(NoopCredentialResolver)
		Expect(isNoop).To(BeFalse(), "explicit resolver must not be replaced by noop")
	})

	It("treats explicit nil resolver as anonymous access", func() {
		client, err := NewOciClientBuilder().WithResolver(nil).Build(ctx)
		Expect(err).NotTo(HaveOccurred())
		ociClient, ok := client.(OciClient)
		Expect(ok).To(BeTrue())
		_, isNoop := ociClient.resolver.(NoopCredentialResolver)
		Expect(isNoop).To(BeTrue())
	})

	It("builds successfully with a logger set", func() {
		client, err := NewOciClientBuilder().WithLogger(logr.Discard()).Build(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(client).NotTo(BeNil())
	})

	It("builds distinct client instances on repeated calls", func() {
		b := NewOciClientBuilder()
		c1, err1 := b.Build(ctx)
		c2, err2 := b.Build(ctx)
		Expect(err1).NotTo(HaveOccurred())
		Expect(err2).NotTo(HaveOccurred())
		Expect(c1).NotTo(BeIdenticalTo(c2))
	})
})

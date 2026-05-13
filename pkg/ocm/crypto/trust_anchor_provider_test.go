package crypto

import (
	"context"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("ConfigMapTrustAnchorProvider", func() {
	var (
		cfg = types.NamespacedName{Namespace: "default", Name: "test-trust-anchor"}
	)

	It("creates provider with default options", func() {
		provider := NewConfigMapTrustAnchorProvider(cfg)
		Expect(provider).ToNot(BeNil())
		Expect(provider.cfg).To(Equal(cfg))
	})

	It("applies WithTrustAnchorProviderLogger option", func() {
		customLog := logr.Discard().WithName("custom")
		provider := NewConfigMapTrustAnchorProvider(cfg, WithTrustAnchorProviderLogger(customLog))
		Expect(provider).ToNot(BeNil())
	})

	It("applies WithNamedTrustAnchorProviderLogger option", func() {
		provider := NewConfigMapTrustAnchorProvider(cfg, WithNamedTrustAnchorProviderLogger(logr.Discard()))
		Expect(provider).ToNot(BeNil())
	})

	It("Get returns nil when data not loaded", func() {
		provider := &ConfigMapTrustAnchorProvider{done: make(chan struct{})}
		creds, err := provider.Get(context.Background())
		Expect(err).ToNot(HaveOccurred())
		Expect(creds).To(BeNil())
	})

	It("Get returns stored credentials", func() {
		provider := &ConfigMapTrustAnchorProvider{done: make(chan struct{})}
		provider.data.Store(map[string]string{"public_key_pem": "test-cert"})

		creds, err := provider.Get(context.Background())
		Expect(err).ToNot(HaveOccurred())
		Expect(creds).To(HaveKeyWithValue("public_key_pem", "test-cert"))
	})

	It("Get returns context error when context is cancelled", func() {
		provider := &ConfigMapTrustAnchorProvider{done: make(chan struct{})}
		cancelledCtx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := provider.Get(cancelledCtx)
		Expect(err).To(Equal(context.Canceled))
	})

	It("Get returns error when provider is stopped", func() {
		provider := &ConfigMapTrustAnchorProvider{done: make(chan struct{})}
		close(provider.done)

		_, err := provider.Get(context.Background())
		Expect(err).To(MatchError("provider stopped"))
	})

	It("SetupWithManager returns error on double call", func() {
		provider := &ConfigMapTrustAnchorProvider{}
		provider.started.Store(true)

		err := provider.SetupWithManager(context.Background(), nil)
		Expect(err).To(MatchError("provider already started"))
	})

	It("ingest returns error for nil configmap", func() {
		provider := &ConfigMapTrustAnchorProvider{}
		err := provider.ingest(nil)
		Expect(err).To(MatchError("configmap is nil"))
	})

	It("ingest returns error when tls.crt key is missing", func() {
		provider := &ConfigMapTrustAnchorProvider{}
		cm := &v1.ConfigMap{Data: map[string]string{"other-key": "value"}}
		err := provider.ingest(cm)
		Expect(err).To(MatchError("key 'tls.crt' not found"))
	})

	It("ingest stores public_key_pem from tls.crt", func() {
		provider := &ConfigMapTrustAnchorProvider{}
		cm := &v1.ConfigMap{Data: map[string]string{"tls.crt": "test-cert-data"}}

		err := provider.ingest(cm)
		Expect(err).ToNot(HaveOccurred())

		stored := provider.data.Load().(map[string]string)
		Expect(stored).To(HaveLen(1))
		Expect(stored["public_key_pem"]).To(Equal("test-cert-data"))
	})

	It("refresh ignores configmaps with non-matching namespace or name", func() {
		provider := &ConfigMapTrustAnchorProvider{cfg: cfg, log: logr.Discard()}
		provider.data.Store(map[string]string{"public_key_pem": "initial"})

		wrongCm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Namespace: "other", Name: cfg.Name},
			Data:       map[string]string{"tls.crt": "wrong"},
		}
		provider.refresh(wrongCm)

		stored := provider.data.Load().(map[string]string)
		Expect(stored["public_key_pem"]).To(Equal("initial"))
	})

	It("refresh updates credentials on matching configmap", func() {
		provider := &ConfigMapTrustAnchorProvider{cfg: cfg, log: logr.Discard()}
		provider.data.Store(map[string]string{"public_key_pem": "initial"})

		updatedCm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Namespace: cfg.Namespace, Name: cfg.Name},
			Data:       map[string]string{"tls.crt": "updated-cert"},
		}
		provider.refresh(updatedCm)

		stored := provider.data.Load().(map[string]string)
		Expect(stored["public_key_pem"]).To(Equal("updated-cert"))
	})

	It("refresh ignores non-configmap objects", func() {
		provider := &ConfigMapTrustAnchorProvider{cfg: cfg, log: logr.Discard()}
		provider.data.Store(map[string]string{"public_key_pem": "initial"})

		provider.refresh(&v1.Secret{})

		stored := provider.data.Load().(map[string]string)
		Expect(stored["public_key_pem"]).To(Equal("initial"))
	})

	It("refresh keeps old credentials when ingest fails", func() {
		provider := &ConfigMapTrustAnchorProvider{cfg: cfg, log: logr.Discard()}
		provider.data.Store(map[string]string{"public_key_pem": "initial"})

		badCm := &v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Namespace: cfg.Namespace, Name: cfg.Name},
			Data:       map[string]string{"wrong-key": "data"},
		}
		provider.refresh(badCm)

		stored := provider.data.Load().(map[string]string)
		Expect(stored["public_key_pem"]).To(Equal("initial"))
	})
})

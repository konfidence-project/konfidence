package crypto

import (
	"context"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("SecretSigningCredentialsProvider", func() {
	var (
		secret = types.NamespacedName{Namespace: "default", Name: "test-signing-secret"}
	)

	It("creates provider with default options", func() {
		provider := NewSecretSigningCredentialsProvider(secret)
		Expect(provider).ToNot(BeNil())
		Expect(provider.secret).To(Equal(secret))
	})

	It("applies WithSigningProviderLogger option", func() {
		customLog := logr.Discard().WithName("custom")
		provider := NewSecretSigningCredentialsProvider(secret, WithSigningProviderLogger(customLog))
		Expect(provider).ToNot(BeNil())
	})

	It("applies WithNamedSigningProviderLogger option", func() {
		provider := NewSecretSigningCredentialsProvider(secret, WithNamedSigningProviderLogger(logr.Discard()))
		Expect(provider).ToNot(BeNil())
	})

	It("Get returns nil when data not loaded", func() {
		provider := &SecretSigningCredentialsProvider{done: make(chan struct{})}
		creds, err := provider.Get(context.Background())
		Expect(err).ToNot(HaveOccurred())
		Expect(creds).To(BeNil())
	})

	It("Get returns stored credentials", func() {
		provider := &SecretSigningCredentialsProvider{done: make(chan struct{})}
		provider.data.Store(map[string]string{"public_key_pem": "cert", "private_key_pem": "key"})

		creds, err := provider.Get(context.Background())
		Expect(err).ToNot(HaveOccurred())
		Expect(creds).To(HaveKeyWithValue("public_key_pem", "cert"))
		Expect(creds).To(HaveKeyWithValue("private_key_pem", "key"))
	})

	It("Get returns context error when context is cancelled", func() {
		provider := &SecretSigningCredentialsProvider{done: make(chan struct{})}
		cancelledCtx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := provider.Get(cancelledCtx)
		Expect(err).To(Equal(context.Canceled))
	})

	It("Get returns error when provider is stopped", func() {
		provider := &SecretSigningCredentialsProvider{done: make(chan struct{})}
		close(provider.done)

		_, err := provider.Get(context.Background())
		Expect(err).To(MatchError("provider stopped"))
	})

	It("SetupWithManager returns error on double call", func() {
		provider := &SecretSigningCredentialsProvider{}
		provider.started.Store(true)

		err := provider.SetupWithManager(context.Background(), nil)
		Expect(err).To(MatchError("provider already started"))
	})

	It("ingest returns error for nil secret", func() {
		provider := &SecretSigningCredentialsProvider{}
		err := provider.ingest(nil)
		Expect(err).To(MatchError("secret is nil"))
	})

	It("ingest returns error when tls.crt is missing", func() {
		provider := &SecretSigningCredentialsProvider{}
		sec := &v1.Secret{Data: map[string][]byte{"tls.key": []byte("key")}}
		err := provider.ingest(sec)
		Expect(err).To(MatchError("keys 'tls.crt' and/or 'tls.key' not found"))
	})

	It("ingest returns error when tls.key is missing", func() {
		provider := &SecretSigningCredentialsProvider{}
		sec := &v1.Secret{Data: map[string][]byte{"tls.crt": []byte("cert")}}
		err := provider.ingest(sec)
		Expect(err).To(MatchError("keys 'tls.crt' and/or 'tls.key' not found"))
	})

	It("ingest stores both public_key_pem and private_key_pem", func() {
		provider := &SecretSigningCredentialsProvider{}
		sec := &v1.Secret{Data: map[string][]byte{
			"tls.crt": []byte("cert-data"),
			"tls.key": []byte("key-data"),
		}}

		err := provider.ingest(sec)
		Expect(err).ToNot(HaveOccurred())

		stored := provider.data.Load().(map[string]string)
		Expect(stored).To(HaveLen(2))
		Expect(stored["public_key_pem"]).To(Equal("cert-data"))
		Expect(stored["private_key_pem"]).To(Equal("key-data"))
	})

	It("refresh ignores secrets with non-matching namespace or name", func() {
		provider := &SecretSigningCredentialsProvider{secret: secret, log: logr.Discard()}
		provider.data.Store(map[string]string{"public_key_pem": "initial", "private_key_pem": "initial"})

		wrongSec := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{Namespace: "other", Name: secret.Name},
			Data:       map[string][]byte{"tls.crt": []byte("wrong"), "tls.key": []byte("wrong")},
		}
		provider.refresh(wrongSec)

		stored := provider.data.Load().(map[string]string)
		Expect(stored["public_key_pem"]).To(Equal("initial"))
	})

	It("refresh updates credentials on matching secret", func() {
		provider := &SecretSigningCredentialsProvider{secret: secret, log: logr.Discard()}
		provider.data.Store(map[string]string{"public_key_pem": "initial", "private_key_pem": "initial"})

		updatedSec := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{Namespace: secret.Namespace, Name: secret.Name},
			Data:       map[string][]byte{"tls.crt": []byte("new-cert"), "tls.key": []byte("new-key")},
		}
		provider.refresh(updatedSec)

		stored := provider.data.Load().(map[string]string)
		Expect(stored["public_key_pem"]).To(Equal("new-cert"))
		Expect(stored["private_key_pem"]).To(Equal("new-key"))
	})

	It("refresh ignores non-secret objects", func() {
		provider := &SecretSigningCredentialsProvider{secret: secret, log: logr.Discard()}
		provider.data.Store(map[string]string{"public_key_pem": "initial", "private_key_pem": "initial"})

		provider.refresh(&v1.ConfigMap{})

		stored := provider.data.Load().(map[string]string)
		Expect(stored["public_key_pem"]).To(Equal("initial"))
	})

	It("refresh keeps old credentials when ingest fails", func() {
		provider := &SecretSigningCredentialsProvider{secret: secret, log: logr.Discard()}
		provider.data.Store(map[string]string{"public_key_pem": "initial", "private_key_pem": "initial"})

		badSec := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{Namespace: secret.Namespace, Name: secret.Name},
			Data:       map[string][]byte{"tls.crt": []byte("cert")}, // missing tls.key
		}
		provider.refresh(badSec)

		stored := provider.data.Load().(map[string]string)
		Expect(stored["public_key_pem"]).To(Equal("initial"))
		Expect(stored["private_key_pem"]).To(Equal("initial"))
	})
})

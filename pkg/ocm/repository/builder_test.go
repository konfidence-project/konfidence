package repository

import (
	"context"
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	genericv1 "ocm.software/open-component-model/bindings/go/configuration/generic/v1/spec"
	credentialsv1 "ocm.software/open-component-model/bindings/go/credentials/spec/config/v1"
	"ocm.software/open-component-model/bindings/go/runtime"
	"ocm.software/open-component-model/kubernetes/controller/pkg/configuration"
)

var _ = Describe("OciClientBuilder", func() {
	var (
		ctx     context.Context
		builder *OciClientBuilder
	)

	BeforeEach(func() {
		ctx = context.Background()
		builder = NewOciClientBuilder()
	})

	Describe("NewOciClientBuilder", func() {
		It("creates a new builder instance", func() {
			b := NewOciClientBuilder()
			Expect(b).ToNot(BeNil())
			Expect(b.log.IsZero()).To(BeTrue())
		})
	})

	Describe("WithLogger", func() {
		It("sets the logger", func() {
			customLog := logr.Discard().WithName("custom")
			result := builder.WithLogger(customLog)

			Expect(result).To(Equal(builder), "should return the builder for chaining")
			Expect(result.log).To(Equal(customLog))
		})
	})

	Describe("WithOCMConfig", func() {
		It("sets the OCM configuration", func() {
			cfg := &configuration.Configuration{}
			result := builder.WithOCMConfig(cfg)

			Expect(result).To(Equal(builder), "should return the builder for chaining")
			Expect(result.ocmConfig).To(Equal(cfg))
		})
	})

	Describe("WithDockerConfigJsonSecret", func() {
		It("sets the secret", func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-secret",
				},
				Type: corev1.SecretTypeDockerConfigJson,
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"registry.example.com":{"auth":"dXNlcjpwYXNz"}}}`),
				},
			}

			result := builder.WithDockerConfigJsonSecret(secret)

			Expect(result).To(Equal(builder), "should return the builder for chaining")
			Expect(result.secret).To(Equal(secret))
		})

		It("allows chaining with WithLogger", func() {
			secret := &corev1.Secret{
				Type: corev1.SecretTypeDockerConfigJson,
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: []byte(`{"auths":{"registry.example.com":{"auth":"dXNlcjpwYXNz"}}}`),
				},
			}

			result := builder.
				WithLogger(logr.Discard()).
				WithDockerConfigJsonSecret(secret)

			Expect(result).To(Equal(builder))
			Expect(result.secret).To(Equal(secret))
		})
	})

	Describe("Build", func() {
		Context("without authentication", func() {
			It("builds a client successfully", func() {
				client, err := builder.Build(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(client).ToNot(BeNil())
				ociClient, ok := client.(OciClient)
				Expect(ok).To(BeTrue(), "built client should be of type *OciClient")
				Expect(ociClient.provider).ToNot(BeNil())
				Expect(ociClient.resolver).ToNot(BeNil())
				Expect(ociClient.log.IsZero()).To(BeTrue(), "default logger should be zero value (discard)")
			})

			It("builds a client with custom logger", func() {
				customLog := logr.Discard().WithName("test")
				c, err := builder.WithLogger(customLog).Build(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(c).ToNot(BeNil())
				ociClient, ok := c.(OciClient)
				Expect(ok).To(BeTrue(), "built client should be of type *OciClient")
				Expect(ociClient.log).To(Equal(customLog))
			})

			It("builds a client with default logger when no logger provided", func() {
				c, err := builder.Build(ctx)
				Expect(err).ToNot(HaveOccurred())
				Expect(c).ToNot(BeNil())
				ociClient, ok := c.(OciClient)
				Expect(ok).To(BeTrue(), "built client should be of type *OciClient")
				Expect(ociClient.log.IsZero()).To(BeTrue(), "default logger should be zero value (discard)")
			})
		})

		Context("with OCM configuration", func() {
			It("builds a client with credential resolver from valid OCM config", func() {
				credScheme := runtime.NewScheme()
				credentialsv1.MustRegister(credScheme)

				// DockerConfig repository entry with sample credentials
				repoRaw := &runtime.Raw{
					Data: []byte(`{
						"type": "DockerConfig/v1",
						"dockerConfig": "{\"auths\":{\"ghcr.io\":{\"auth\":\"dGVzdDp0ZXN0\"}}}"
					}`),
				}

				credConfig := &credentialsv1.Config{
					Repositories: []credentialsv1.RepositoryConfigEntry{{Repository: repoRaw}},
				}
				rawCreds := &runtime.Raw{}
				err := credScheme.Convert(credConfig, rawCreds)
				Expect(err).ToNot(HaveOccurred())

				cfg := &genericv1.Config{
					Type:           runtime.Type{Version: genericv1.Version, Name: genericv1.ConfigType},
					Configurations: []*runtime.Raw{rawCreds},
				}

				ocmCfg := &configuration.Configuration{Config: cfg}

				client, err := NewOciClientBuilder().
					WithOCMConfig(ocmCfg).
					Build(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(client).ToNot(BeNil())
				ociClient, ok := client.(OciClient)
				Expect(ok).To(BeTrue())
				Expect(ociClient.resolver).ToNot(BeNil())
				_, isNoop := ociClient.resolver.(NoopCredentialResolver)
				Expect(isNoop).To(BeFalse(), "resolver should not be noop when OCM config with credentials is provided")
			})

			It("returns error when no credential configuration is found in OCM config", func() {
				cfg := &genericv1.Config{
					Type:           runtime.Type{Version: genericv1.Version, Name: genericv1.ConfigType},
					Configurations: []*runtime.Raw{{Data: []byte(`{"type":"unrelated.config/v1"}`)}},
				}

				ocmCfg := &configuration.Configuration{Config: cfg}

				client, err := NewOciClientBuilder().
					WithOCMConfig(ocmCfg).
					Build(ctx)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no credential configuration found"))
				Expect(client).To(BeNil())
			})

			It("returns error when credential graph building fails", func() {
				credScheme := runtime.NewScheme()
				credentialsv1.MustRegister(credScheme)

				// Unknown repository type that LookupCredentialConfig can parse but buildGraph cannot resolve
				unknownRepo := &runtime.Raw{
					Data: []byte(`{"type":"unknown.repository.type/v1"}`),
				}

				credConfig := &credentialsv1.Config{
					Repositories: []credentialsv1.RepositoryConfigEntry{{Repository: unknownRepo}},
				}
				rawCreds := &runtime.Raw{}
				err := credScheme.Convert(credConfig, rawCreds)
				Expect(err).ToNot(HaveOccurred())

				cfg := &genericv1.Config{
					Type:           runtime.Type{Version: genericv1.Version, Name: genericv1.ConfigType},
					Configurations: []*runtime.Raw{rawCreds},
				}

				ocmCfg := &configuration.Configuration{Config: cfg}

				client, err := NewOciClientBuilder().
					WithOCMConfig(ocmCfg).
					Build(ctx)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("building credential graph"))
				Expect(client).To(BeNil())
			})
		})

		Context("with docker config secret (deprecated)", func() {
			var validDockerConfig map[string]interface{}

			BeforeEach(func() {
				validDockerConfig = map[string]interface{}{
					"auths": map[string]interface{}{
						"ghcr.io": map[string]interface{}{
							"auth": "dGVzdDp0ZXN0", // base64("test:test")
						},
						"registry.example.com": map[string]interface{}{
							"username": "user",
							"password": "pass",
						},
					},
				}
			})

			It("builds a client with credential resolver from valid secret", func() {
				dockerConfigJSON, err := json.Marshal(validDockerConfig)
				Expect(err).ToNot(HaveOccurred())

				secret := &corev1.Secret{
					Type: corev1.SecretTypeDockerConfigJson,
					Data: map[string][]byte{
						corev1.DockerConfigJsonKey: dockerConfigJSON,
					},
				}

				client, err := builder.
					WithDockerConfigJsonSecret(secret).
					Build(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(client).ToNot(BeNil())
				ociClient, ok := client.(OciClient)
				Expect(ok).To(BeTrue())
				Expect(ociClient.resolver).ToNot(BeNil())
				_, isNoop := ociClient.resolver.(NoopCredentialResolver)
				Expect(isNoop).To(BeFalse(), "resolver should not be noop when secret with credentials is provided")
			})

			It("returns error when secret is missing .dockerconfigjson key", func() {
				secret := &corev1.Secret{
					Type: corev1.SecretTypeDockerConfigJson,
					Data: map[string][]byte{
						"wrong-key": []byte(`{}`),
					},
				}

				client, err := builder.
					WithDockerConfigJsonSecret(secret).
					Build(ctx)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("does not contain key"))
				Expect(err.Error()).To(ContainSubstring(corev1.DockerConfigJsonKey))
				Expect(client).To(BeNil())
			})

			It("handles secret with empty auths object", func() {
				emptyAuthsConfig := map[string]interface{}{
					"auths": map[string]interface{}{},
				}
				dockerConfigJSON, err := json.Marshal(emptyAuthsConfig)
				Expect(err).ToNot(HaveOccurred())

				secret := &corev1.Secret{
					Type: corev1.SecretTypeDockerConfigJson,
					Data: map[string][]byte{
						corev1.DockerConfigJsonKey: dockerConfigJSON,
					},
				}

				client, err := builder.
					WithDockerConfigJsonSecret(secret).
					Build(ctx)

				Expect(err).ToNot(HaveOccurred())
				Expect(client).ToNot(BeNil())
				ociClient, ok := client.(OciClient)
				Expect(ok).To(BeTrue(), "built client should be of type *OciClient")
				Expect(ociClient.resolver).ToNot(BeNil())
			})
		})

		Context("builder reuse", func() {
			It("can build multiple clients from the same builder", func() {
				client1, err1 := builder.Build(ctx)
				Expect(err1).ToNot(HaveOccurred())

				client2, err2 := builder.Build(ctx)
				Expect(err2).ToNot(HaveOccurred())

				// Clients should be independent instances
				Expect(client1).ToNot(BeIdenticalTo(client2))
			})

			It("reflects configuration changes between builds", func() {
				c1, err := builder.Build(ctx)
				Expect(err).ToNot(HaveOccurred())

				// Modify builder and build again
				customLog := logr.Discard().WithName("custom")
				c2, err := builder.WithLogger(customLog).Build(ctx)
				Expect(err).ToNot(HaveOccurred())
				// Both clients should be created successfully with different configs
				Expect(c1).ToNot(BeNil())
				Expect(c2).ToNot(BeNil())
				ociClient1, ok1 := c1.(OciClient)
				ociClient2, ok2 := c2.(OciClient)
				Expect(ok1).To(BeTrue())
				Expect(ok2).To(BeTrue())
				Expect(ociClient1.log.IsZero()).To(BeTrue(), "first client should have default logger")
				Expect(ociClient2.log).To(Equal(customLog), "second client should have custom logger")
				Expect(ociClient1.resolver).ToNot(BeNil())
				Expect(ociClient1.provider).ToNot(BeNil())
				Expect(ociClient2.resolver).ToNot(BeNil())
				Expect(ociClient2.provider).ToNot(BeNil())
				Expect(ociClient1).ToNot(BeIdenticalTo(ociClient2), "clients should be different instances")
			})
		})
	})

	Describe("Integration scenarios", func() {
		It("supports typical Kubernetes operator pattern with docker config secret (deprecated)", func() {
			dockerConfig := map[string]interface{}{
				"auths": map[string]interface{}{
					"ghcr.io": map[string]interface{}{
						"auth": "dGVzdDp0ZXN0",
					},
				},
			}
			dockerConfigJSON, err := json.Marshal(dockerConfig)
			Expect(err).ToNot(HaveOccurred())

			pullSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ghcr-pull-secret",
					Namespace: "default",
				},
				Type: corev1.SecretTypeDockerConfigJson,
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: dockerConfigJSON,
				},
			}

			client, err := NewOciClientBuilder().
				WithDockerConfigJsonSecret(pullSecret).
				WithLogger(logr.Discard().WithName("ocm-operator")).
				Build(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(client).ToNot(BeNil())
		})

		It("supports public registry access without authentication", func() {
			// Common case: accessing public registries
			client, err := NewOciClientBuilder().
				WithLogger(logr.Discard().WithName("public-access")).
				Build(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(client).ToNot(BeNil())
			// Client should work for public registries without credentials
		})
	})
})

package repository

import (
	"context"
	"encoding/json"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("OciClientBuilder", func() {
	var (
		ctx     context.Context
		builder *OciClientBuilder
		log     logr.Logger
	)

	BeforeEach(func() {
		ctx = context.Background()
		builder = NewOciClientBuilder()
		log = logr.Discard()
	})

	Describe("NewOciClientBuilder", func() {
		It("creates a new builder instance", func() {
			b := NewOciClientBuilder()
			Expect(b).ToNot(BeNil())
			Expect(b.log.IsZero()).To(BeTrue())
			Expect(b.secret).To(BeNil())
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
				WithLogger(log).
				WithDockerConfigJsonSecret(secret)

			Expect(result).To(Equal(builder))
			Expect(result.log).To(Equal(log))
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

		Context("with authentication", func() {
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
				Expect(ok).To(BeTrue(), "built client should be of type *OciClient")
				Expect(ociClient.resolver).ToNot(BeNil())
				Expect(ociClient.provider).ToNot(BeNil())
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

				// Should succeed - empty auths is valid Docker config
				Expect(err).ToNot(HaveOccurred())
				Expect(client).ToNot(BeNil())
				ociClient, ok := client.(OciClient)
				Expect(ok).To(BeTrue(), "built client should be of type *OciClient")
				Expect(ociClient.resolver).ToNot(BeNil())
				Expect(ociClient.provider).ToNot(BeNil())
				Expect(ociClient.log.IsZero()).To(BeTrue(), "default logger should be zero value (discard)")
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
		It("supports typical Kubernetes operator pattern", func() {
			// Simulate a Kubernetes operator fetching a pull secret
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

			// Build client with operator logger pattern
			operatorLog := logr.Discard().WithName("ocm-operator")
			client, err := NewOciClientBuilder().
				WithDockerConfigJsonSecret(pullSecret).
				WithLogger(operatorLog).
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

		It("supports multi-registry authentication", func() {
			// Real-world scenario: multiple registries in one config
			dockerConfig := map[string]interface{}{
				"auths": map[string]interface{}{
					"ghcr.io": map[string]interface{}{
						"username": "github-user",
						"password": "ghp_token",
					},
					"docker.io": map[string]interface{}{
						"auth": "ZG9ja2VyOnRva2Vu", // base64("docker:token")
					},
					"registry.example.com": map[string]interface{}{
						"username": "company-user",
						"password": "company-pass",
					},
				},
			}
			dockerConfigJSON, err := json.Marshal(dockerConfig)
			Expect(err).ToNot(HaveOccurred())

			secret := &corev1.Secret{
				Type: corev1.SecretTypeDockerConfigJson,
				Data: map[string][]byte{
					corev1.DockerConfigJsonKey: dockerConfigJSON,
				},
			}

			client, err := NewOciClientBuilder().
				WithDockerConfigJsonSecret(secret).
				Build(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(client).ToNot(BeNil())
		})
	})
})

package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-logr/logr"
	konfcompref "github.com/konfidence-project/pkg/ocm/compref"
	"github.com/konfidence-project/pkg/ocm/repository/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"ocm.software/open-component-model/bindings/go/credentials"
	descruntime "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	ocispec "ocm.software/open-component-model/bindings/go/oci/spec/repository/v1/oci"
	"ocm.software/open-component-model/bindings/go/repository"
	"ocm.software/open-component-model/bindings/go/runtime"
)

var _ = Describe("OciClient", func() {
	var (
		ctx                  context.Context
		log                  logr.Logger
		mockCtrl             *gomock.Controller
		resolverMock         *mocks.MockResolver
		providerMock         *mocks.MockComponentVersionRepositoryProvider
		repoMock             *mocks.MockComponentVersionRepository
		transferExecutorMock *mocks.MockTransferExecutor
		client               OciClient
	)

	// Helper function to create a descriptor with the correct nested structure
	makeDescriptor := func(name, version string) descruntime.Descriptor {
		return descruntime.Descriptor{
			Component: descruntime.Component{
				ComponentMeta: descruntime.ComponentMeta{
					ObjectMeta: descruntime.ObjectMeta{
						Name:    name,
						Version: version,
					},
				},
				Provider: descruntime.Provider{Name: "acme"},
			},
		}
	}

	// makeOCIRepoSpec creates a properly typed OCI repository spec that the OCM
	// transfer library accepts (unlike *runtime.Unstructured which is rejected).
	makeOCIRepoSpec := func(baseUrl string) *ocispec.Repository {
		return &ocispec.Repository{
			Type:    runtime.Type{Name: ocispec.Type, Version: "v1"},
			BaseUrl: baseUrl,
		}
	}

	BeforeEach(func() {
		ctx = context.Background()
		log = logr.Discard()
		mockCtrl = gomock.NewController(GinkgoT())
		resolverMock = mocks.NewMockResolver(mockCtrl)
		providerMock = mocks.NewMockComponentVersionRepositoryProvider(mockCtrl)
		repoMock = mocks.NewMockComponentVersionRepository(mockCtrl)
		transferExecutorMock = mocks.NewMockTransferExecutor(mockCtrl)

		client = NewOciClient(resolverMock, providerMock, transferExecutorMock, WithOciClientLogger(log))
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("NewOciClient", func() {
		It("creates a client successfully", func() {
			c := NewOciClient(resolverMock, providerMock, transferExecutorMock)
			Expect(c).ToNot(BeNil())
		})

		It("applies logger option", func() {
			customLog := logr.Discard().WithName("custom")
			c := NewOciClient(resolverMock, providerMock, transferExecutorMock, WithOciClientLogger(customLog))

			Expect(c).ToNot(BeNil())
			Expect(c.log).To(Equal(customLog))
		})
	})

	Describe("Get", func() {
		var (
			repoSpec *runtime.Unstructured
			identity runtime.Identity
			creds    map[string]string
		)

		BeforeEach(func() {
			repoSpec = &runtime.Unstructured{
				Data: map[string]interface{}{
					"type": "oci",
				},
			}
			identity = runtime.Identity{"type": "ociRegistry", "hostname": "ghcr.io"}
			creds = map[string]string{"username": "user", "password": "pass"}
		})

		Context("with specific version", func() {
			It("retrieves a component descriptor successfully", func() {
				ref := compref.Ref{
					Repository: repoSpec,
					Component:  "github.com/acme/backend",
					Version:    "1.0.0",
				}
				expectedDesc := descruntime.Descriptor{
					Component: descruntime.Component{
						ComponentMeta: descruntime.ComponentMeta{
							ObjectMeta: descruntime.ObjectMeta{
								Name:    "github.com/acme/backend",
								Version: "1.0.0",
							},
						},
					},
				}

				providerMock.EXPECT().
					GetComponentVersionRepositoryCredentialConsumerIdentity(gomock.Any(), repoSpec).
					Return(identity, nil)
				resolverMock.EXPECT().
					Resolve(gomock.Any(), identity).
					Return(creds, nil)
				providerMock.EXPECT().
					GetComponentVersionRepository(gomock.Any(), repoSpec, creds).
					Return(repoMock, nil)
				repoMock.EXPECT().
					GetComponentVersion(gomock.Any(), "github.com/acme/backend", "1.0.0").
					Return(&expectedDesc, nil)

				desc, err := client.Get(ctx, ref)

				Expect(err).ToNot(HaveOccurred())
				Expect(desc).To(Equal(expectedDesc))
			})

			It("returns ErrNotFound when component version doesn't exist", func() {
				ref := compref.Ref{
					Repository: repoSpec,
					Component:  "github.com/acme/missing",
					Version:    "1.0.0",
				}

				providerMock.EXPECT().
					GetComponentVersionRepositoryCredentialConsumerIdentity(gomock.Any(), repoSpec).
					Return(identity, nil)
				resolverMock.EXPECT().
					Resolve(gomock.Any(), identity).
					Return(creds, nil)
				providerMock.EXPECT().
					GetComponentVersionRepository(gomock.Any(), repoSpec, creds).
					Return(repoMock, nil)
				repoMock.EXPECT().
					GetComponentVersion(gomock.Any(), "github.com/acme/missing", "1.0.0").
					Return(nil, repository.ErrNotFound)

				desc, err := client.Get(ctx, ref)

				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, ErrNotFound)).To(BeTrue())
				Expect(desc).To(Equal(descruntime.Descriptor{}))
			})

			It("works with anonymous access when no credentials found", func() {
				ref := compref.Ref{
					Repository: repoSpec,
					Component:  "github.com/acme/public",
					Version:    "1.0.0",
				}
				expectedDesc := makeDescriptor("github.com/acme/public", "1.0.0")

				providerMock.EXPECT().
					GetComponentVersionRepositoryCredentialConsumerIdentity(gomock.Any(), repoSpec).
					Return(identity, nil)
				resolverMock.EXPECT().
					Resolve(gomock.Any(), identity).
					Return(nil, fmt.Errorf("credentials: %w", credentials.ErrNotFound))
				providerMock.EXPECT().
					GetComponentVersionRepository(gomock.Any(), repoSpec, gomock.Nil()).
					Return(repoMock, nil)
				repoMock.EXPECT().
					GetComponentVersion(gomock.Any(), "github.com/acme/public", "1.0.0").
					Return(&expectedDesc, nil)

				desc, err := client.Get(ctx, ref)

				Expect(err).ToNot(HaveOccurred())
				Expect(desc).To(Equal(expectedDesc))
			})

			It("returns error when repository access fails", func() {
				ref := compref.Ref{
					Repository: repoSpec,
					Component:  "github.com/acme/backend",
					Version:    "1.0.0",
				}

				providerMock.EXPECT().
					GetComponentVersionRepositoryCredentialConsumerIdentity(gomock.Any(), repoSpec).
					Return(identity, nil)
				resolverMock.EXPECT().
					Resolve(gomock.Any(), identity).
					Return(creds, nil)
				providerMock.EXPECT().
					GetComponentVersionRepository(gomock.Any(), repoSpec, creds).
					Return(nil, fmt.Errorf("connection failed"))

				desc, err := client.Get(ctx, ref)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("connection failed"))
				Expect(desc).To(Equal(descruntime.Descriptor{}))
			})

			It("returns error when credential identity lookup fails", func() {
				ref := compref.Ref{
					Repository: repoSpec,
					Component:  "github.com/acme/backend",
					Version:    "1.0.0",
				}

				providerMock.EXPECT().
					GetComponentVersionRepositoryCredentialConsumerIdentity(gomock.Any(), repoSpec).
					Return(nil, fmt.Errorf("invalid repo spec"))

				desc, err := client.Get(ctx, ref)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid repo spec"))
				Expect(desc).To(Equal(descruntime.Descriptor{}))
			})

			Context("with version alias", func() {
				It("retrieves a component descriptor using an alias", func() {
					ref := compref.Ref{
						Repository: repoSpec,
						Component:  "github.com/acme/backend",
						Version:    "latest", // Using alias instead of semantic version
					}
					expectedDesc := makeDescriptor("github.com/acme/backend", "1.2.3")

					providerMock.EXPECT().
						GetComponentVersionRepositoryCredentialConsumerIdentity(gomock.Any(), repoSpec).
						Return(identity, nil)
					resolverMock.EXPECT().
						Resolve(gomock.Any(), identity).
						Return(creds, nil)
					providerMock.EXPECT().
						GetComponentVersionRepository(gomock.Any(), repoSpec, creds).
						Return(repoMock, nil)
					repoMock.EXPECT().
						GetComponentVersion(gomock.Any(), "github.com/acme/backend", "latest").
						Return(&expectedDesc, nil)

					desc, err := client.Get(ctx, ref)

					Expect(err).ToNot(HaveOccurred())
					Expect(desc).To(Equal(expectedDesc))
				})

				It("returns ErrNotFound when alias doesn't exist", func() {
					ref := compref.Ref{
						Repository: repoSpec,
						Component:  "github.com/acme/backend",
						Version:    "nonexistent-alias",
					}

					providerMock.EXPECT().
						GetComponentVersionRepositoryCredentialConsumerIdentity(gomock.Any(), repoSpec).
						Return(identity, nil)
					resolverMock.EXPECT().
						Resolve(gomock.Any(), identity).
						Return(creds, nil)
					providerMock.EXPECT().
						GetComponentVersionRepository(gomock.Any(), repoSpec, creds).
						Return(repoMock, nil)
					repoMock.EXPECT().
						GetComponentVersion(gomock.Any(), "github.com/acme/backend", "nonexistent-alias").
						Return(nil, repository.ErrNotFound)

					desc, err := client.Get(ctx, ref)

					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, ErrNotFound)).To(BeTrue())
					Expect(desc).To(Equal(descruntime.Descriptor{}))
				})
			})
		})

		Context("reference validation", func() {
			It("returns error when version is empty", func() {
				ref := compref.Ref{
					Repository: repoSpec,
					Component:  "github.com/acme/backend",
					Version:    "",
				}

				desc, err := client.Get(ctx, ref)

				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, ErrInvalidComponentReference)).To(BeTrue())
				Expect(err.Error()).To(ContainSubstring("invalid reference for Get"))
				Expect(desc).To(Equal(descruntime.Descriptor{}))
			})
		})
	})

	Describe("Save", func() {
		var (
			repoSpec *runtime.Unstructured
			identity runtime.Identity
			creds    map[string]string
			desc     descruntime.Descriptor
		)

		BeforeEach(func() {
			repoSpec = &runtime.Unstructured{
				Data: map[string]interface{}{
					"type": "oci",
				},
			}
			identity = runtime.Identity{"type": "ociRegistry", "hostname": "ghcr.io"}
			creds = map[string]string{"username": "user", "password": "pass"}
			desc = descruntime.Descriptor{
				Component: descruntime.Component{
					ComponentMeta: descruntime.ComponentMeta{
						ObjectMeta: descruntime.ObjectMeta{
							Name:    "github.com/acme/backend",
							Version: "1.0.0",
						},
					},
				},
			}
		})

		Context("successful save", func() {
			It("saves a new component descriptor", func() {
				providerMock.EXPECT().
					GetComponentVersionRepositoryCredentialConsumerIdentity(gomock.Any(), repoSpec).
					Return(identity, nil)
				resolverMock.EXPECT().
					Resolve(gomock.Any(), identity).
					Return(creds, nil)
				providerMock.EXPECT().
					GetComponentVersionRepository(gomock.Any(), repoSpec, creds).
					Return(repoMock, nil)
				repoMock.EXPECT().
					GetComponentVersion(gomock.Any(), desc.Component.Name, desc.Component.Version).
					Return(nil, repository.ErrNotFound)
				repoMock.EXPECT().
					AddComponentVersion(gomock.Any(), &desc).
					Return(nil)

				err := client.Save(ctx, repoSpec, desc)

				Expect(err).ToNot(HaveOccurred())
			})

			It("works with anonymous access when no credentials found", func() {
				providerMock.EXPECT().
					GetComponentVersionRepositoryCredentialConsumerIdentity(gomock.Any(), repoSpec).
					Return(identity, nil)
				resolverMock.EXPECT().
					Resolve(gomock.Any(), identity).
					Return(nil, fmt.Errorf("credentials: %w", credentials.ErrNotFound))
				providerMock.EXPECT().
					GetComponentVersionRepository(gomock.Any(), repoSpec, gomock.Nil()).
					Return(repoMock, nil)
				repoMock.EXPECT().
					GetComponentVersion(gomock.Any(), desc.Component.Name, desc.Component.Version).
					Return(nil, repository.ErrNotFound)
				repoMock.EXPECT().
					AddComponentVersion(gomock.Any(), &desc).
					Return(nil)

				err := client.Save(ctx, repoSpec, desc)

				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("duplicate detection", func() {
			It("returns ErrComponentAlreadyExists when component version exists", func() {
				existingDesc := makeDescriptor("github.com/acme/backend", "1.0.0")

				providerMock.EXPECT().
					GetComponentVersionRepositoryCredentialConsumerIdentity(gomock.Any(), repoSpec).
					Return(identity, nil)
				resolverMock.EXPECT().
					Resolve(gomock.Any(), identity).
					Return(creds, nil)
				providerMock.EXPECT().
					GetComponentVersionRepository(gomock.Any(), repoSpec, creds).
					Return(repoMock, nil)
				repoMock.EXPECT().
					GetComponentVersion(gomock.Any(), desc.Component.Name, desc.Component.Version).
					Return(&existingDesc, nil)

				err := client.Save(ctx, repoSpec, desc)

				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, ErrComponentAlreadyExists)).To(BeTrue())
			})
		})

		Context("error handling", func() {
			It("returns error when repository access fails", func() {
				providerMock.EXPECT().
					GetComponentVersionRepositoryCredentialConsumerIdentity(gomock.Any(), repoSpec).
					Return(identity, nil)
				resolverMock.EXPECT().
					Resolve(gomock.Any(), identity).
					Return(creds, nil)
				providerMock.EXPECT().
					GetComponentVersionRepository(gomock.Any(), repoSpec, creds).
					Return(nil, fmt.Errorf("connection failed"))

				err := client.Save(ctx, repoSpec, desc)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("connection failed"))
			})

			It("returns error when credential identity lookup fails", func() {
				providerMock.EXPECT().
					GetComponentVersionRepositoryCredentialConsumerIdentity(gomock.Any(), repoSpec).
					Return(nil, fmt.Errorf("invalid repo spec"))

				err := client.Save(ctx, repoSpec, desc)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid repo spec"))
			})

			It("returns error when AddComponentVersion fails", func() {
				providerMock.EXPECT().
					GetComponentVersionRepositoryCredentialConsumerIdentity(gomock.Any(), repoSpec).
					Return(identity, nil)
				resolverMock.EXPECT().
					Resolve(gomock.Any(), identity).
					Return(creds, nil)
				providerMock.EXPECT().
					GetComponentVersionRepository(gomock.Any(), repoSpec, creds).
					Return(repoMock, nil)
				repoMock.EXPECT().
					GetComponentVersion(gomock.Any(), desc.Component.Name, desc.Component.Version).
					Return(nil, repository.ErrNotFound)
				repoMock.EXPECT().
					AddComponentVersion(gomock.Any(), &desc).
					Return(fmt.Errorf("storage failed"))

				err := client.Save(ctx, repoSpec, desc)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("storage failed"))
			})

			It("returns error when existence check fails with non-NotFound error", func() {
				providerMock.EXPECT().
					GetComponentVersionRepositoryCredentialConsumerIdentity(gomock.Any(), repoSpec).
					Return(identity, nil)
				resolverMock.EXPECT().
					Resolve(gomock.Any(), identity).
					Return(creds, nil)
				providerMock.EXPECT().
					GetComponentVersionRepository(gomock.Any(), repoSpec, creds).
					Return(repoMock, nil)
				repoMock.EXPECT().
					GetComponentVersion(gomock.Any(), desc.Component.Name, desc.Component.Version).
					Return(nil, fmt.Errorf("registry error"))

				err := client.Save(ctx, repoSpec, desc)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("check if component version already exists"))
			})
		})
	})

	Describe("Copy", func() {
		var (
			sourceRepoSpec *runtime.Unstructured
			targetRepoSpec *runtime.Unstructured
			identity       runtime.Identity
			creds          map[string]string
		)

		BeforeEach(func() {
			sourceRepoSpec = &runtime.Unstructured{
				Data: map[string]interface{}{
					"type":    "oci",
					"baseUrl": "ghcr.io/acme/source",
				},
			}
			targetRepoSpec = &runtime.Unstructured{
				Data: map[string]interface{}{
					"type":    "oci",
					"baseUrl": "ghcr.io/acme/target",
				},
			}
			identity = runtime.Identity{"type": "ociRegistry", "hostname": "ghcr.io"}
			creds = map[string]string{"username": "user", "password": "pass"}
		})

		Context("repository resolution errors", func() {
			It("returns error when credential identity lookup fails", func() {
				refs := []compref.Ref{
					{
						Repository: sourceRepoSpec,
						Component:  "github.com/acme/backend",
						Version:    "1.0.0",
					},
				}

				providerMock.EXPECT().
					GetComponentVersionRepositoryCredentialConsumerIdentity(gomock.Any(), sourceRepoSpec).
					Return(nil, fmt.Errorf("invalid repo spec"))

				err := client.Copy(ctx, refs, targetRepoSpec)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("getting repository for component reference"))
				Expect(err.Error()).To(ContainSubstring("invalid repo spec"))
			})

			It("returns error when credential resolution fails with non-NotFound error", func() {
				refs := []compref.Ref{
					{
						Repository: sourceRepoSpec,
						Component:  "github.com/acme/backend",
						Version:    "1.0.0",
					},
				}

				providerMock.EXPECT().
					GetComponentVersionRepositoryCredentialConsumerIdentity(gomock.Any(), sourceRepoSpec).
					Return(identity, nil)
				resolverMock.EXPECT().
					Resolve(gomock.Any(), identity).
					Return(nil, fmt.Errorf("auth service unavailable"))

				err := client.Copy(ctx, refs, targetRepoSpec)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("auth service unavailable"))
			})

			It("fails on second reference when its repository cannot be resolved", func() {
				secondRepoSpec := &runtime.Unstructured{
					Data: map[string]interface{}{
						"type":    "oci",
						"baseUrl": "private.registry.io/components",
					},
				}
				secondIdentity := runtime.Identity{"type": "ociRegistry", "hostname": "private.registry.io"}

				refs := []compref.Ref{
					{
						Repository: sourceRepoSpec,
						Component:  "github.com/acme/backend",
						Version:    "1.0.0",
					},
					{
						Repository: secondRepoSpec,
						Component:  "github.com/acme/frontend",
						Version:    "2.0.0",
					},
				}

				// First reference resolves successfully
				providerMock.EXPECT().
					GetComponentVersionRepositoryCredentialConsumerIdentity(gomock.Any(), sourceRepoSpec).
					Return(identity, nil)
				resolverMock.EXPECT().
					Resolve(gomock.Any(), identity).
					Return(creds, nil)
				providerMock.EXPECT().
					GetComponentVersionRepository(gomock.Any(), sourceRepoSpec, creds).
					Return(repoMock, nil)

				// Second reference fails at credential resolution
				providerMock.EXPECT().
					GetComponentVersionRepositoryCredentialConsumerIdentity(gomock.Any(), secondRepoSpec).
					Return(secondIdentity, nil)
				resolverMock.EXPECT().
					Resolve(gomock.Any(), secondIdentity).
					Return(nil, fmt.Errorf("forbidden"))

				err := client.Copy(ctx, refs, targetRepoSpec)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("github.com/acme/frontend"))
				Expect(err.Error()).To(ContainSubstring("forbidden"))
			})
		})

		Context("reference validation", func() {
			It("returns error when a reference has empty version", func() {
				refs := []compref.Ref{
					{
						Repository: sourceRepoSpec,
						Component:  "github.com/acme/backend",
						Version:    "",
					},
				}

				err := client.Copy(ctx, refs, targetRepoSpec)

				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, ErrInvalidComponentReference)).To(BeTrue())
				Expect(err.Error()).To(ContainSubstring("invalid reference for Copy"))
			})

			It("returns error when second reference in slice has empty version", func() {
				refs := []compref.Ref{
					{
						Repository: sourceRepoSpec,
						Component:  "github.com/acme/backend",
						Version:    "1.0.0",
					},
					{
						Repository: sourceRepoSpec,
						Component:  "github.com/acme/frontend",
						Version:    "",
					},
				}

				// First reference will be validated and processed, so we need mocks
				providerMock.EXPECT().
					GetComponentVersionRepositoryCredentialConsumerIdentity(gomock.Any(), sourceRepoSpec).
					Return(identity, nil)
				resolverMock.EXPECT().
					Resolve(gomock.Any(), identity).
					Return(creds, nil)
				providerMock.EXPECT().
					GetComponentVersionRepository(gomock.Any(), sourceRepoSpec, creds).
					Return(repoMock, nil)

				err := client.Copy(ctx, refs, targetRepoSpec)

				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, ErrInvalidComponentReference)).To(BeTrue())
				Expect(err.Error()).To(ContainSubstring("invalid reference for Copy"))
			})
		})

		Context("transfer graph errors", func() {
			It("returns error when called with empty references", func() {
				err := client.Copy(ctx, []compref.Ref{}, targetRepoSpec)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("building transfer graph definition"))
			})
		})

		Context("transfer execution", func() {
			var (
				sourceOCISpec *ocispec.Repository
				targetOCISpec *ocispec.Repository
			)

			BeforeEach(func() {
				sourceOCISpec = makeOCIRepoSpec("ghcr.io/acme/source")
				targetOCISpec = makeOCIRepoSpec("ghcr.io/acme/target")
			})

			It("executes transfer successfully", func() {
				refs := []compref.Ref{
					{
						Repository: sourceOCISpec,
						Component:  "github.com/acme/backend",
						Version:    "1.0.0",
					},
				}
				providerMock.EXPECT().
					GetComponentVersionRepositoryCredentialConsumerIdentity(gomock.Any(), sourceOCISpec).
					Return(identity, nil)
				resolverMock.EXPECT().
					Resolve(gomock.Any(), identity).
					Return(creds, nil)
				providerMock.EXPECT().
					GetComponentVersionRepository(gomock.Any(), sourceOCISpec, creds).
					Return(repoMock, nil)
				repoMock.EXPECT().
					GetComponentVersion(gomock.Any(), "github.com/acme/backend", "1.0.0").
					Return(new(makeDescriptor("github.com/acme/backend", "1.0.0")), nil)
				transferExecutorMock.EXPECT().
					Execute(gomock.Any(), gomock.Any()).
					Return(nil)

				err := client.Copy(ctx, refs, targetOCISpec)

				Expect(err).ToNot(HaveOccurred())
			})

			It("returns error when transfer execution fails", func() {
				refs := []compref.Ref{
					{
						Repository: sourceOCISpec,
						Component:  "github.com/acme/backend",
						Version:    "1.0.0",
					},
				}
				providerMock.EXPECT().
					GetComponentVersionRepositoryCredentialConsumerIdentity(gomock.Any(), sourceOCISpec).
					Return(identity, nil)
				resolverMock.EXPECT().
					Resolve(gomock.Any(), identity).
					Return(creds, nil)
				providerMock.EXPECT().
					GetComponentVersionRepository(gomock.Any(), sourceOCISpec, creds).
					Return(repoMock, nil)
				repoMock.EXPECT().
					GetComponentVersion(gomock.Any(), "github.com/acme/backend", "1.0.0").
					Return(new(makeDescriptor("github.com/acme/backend", "1.0.0")), nil)
				transferExecutorMock.EXPECT().
					Execute(gomock.Any(), gomock.Any()).
					Return(fmt.Errorf("transfer failed: network timeout"))

				err := client.Copy(ctx, refs, targetOCISpec)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("executing copy"))
				Expect(err.Error()).To(ContainSubstring("transfer failed: network timeout"))
			})

			It("transfers multiple references successfully", func() {
				secondOCISpec := makeOCIRepoSpec("ghcr.io/acme/second-source")
				secondRepoMock := mocks.NewMockComponentVersionRepository(mockCtrl)
				secondIdentity := runtime.Identity{"type": "ociRegistry", "hostname": "ghcr.io"}

				refs := []compref.Ref{
					{
						Repository: sourceOCISpec,
						Component:  "github.com/acme/backend",
						Version:    "1.0.0",
					},
					{
						Repository: secondOCISpec,
						Component:  "github.com/acme/frontend",
						Version:    "2.0.0",
					},
				} // First reference
				providerMock.EXPECT().
					GetComponentVersionRepositoryCredentialConsumerIdentity(gomock.Any(), sourceOCISpec).
					Return(identity, nil)
				resolverMock.EXPECT().
					Resolve(gomock.Any(), identity).
					Return(creds, nil)
				providerMock.EXPECT().
					GetComponentVersionRepository(gomock.Any(), sourceOCISpec, creds).
					Return(repoMock, nil)
				repoMock.EXPECT().
					GetComponentVersion(gomock.Any(), "github.com/acme/backend", "1.0.0").
					Return(new(makeDescriptor("github.com/acme/backend", "1.0.0")), nil)

				// Second reference
				providerMock.EXPECT().
					GetComponentVersionRepositoryCredentialConsumerIdentity(gomock.Any(), secondOCISpec).
					Return(secondIdentity, nil)
				resolverMock.EXPECT().
					Resolve(gomock.Any(), secondIdentity).
					Return(creds, nil)
				providerMock.EXPECT().
					GetComponentVersionRepository(gomock.Any(), secondOCISpec, creds).
					Return(secondRepoMock, nil)
				secondRepoMock.EXPECT().
					GetComponentVersion(gomock.Any(), "github.com/acme/frontend", "2.0.0").
					Return(new(makeDescriptor("github.com/acme/frontend", "2.0.0")), nil)

				transferExecutorMock.EXPECT().
					Execute(gomock.Any(), gomock.Any()).
					Return(nil)

				err := client.Copy(ctx, refs, targetOCISpec)

				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("AddAlias", func() {
		var (
			repoSpec *runtime.Unstructured
			identity runtime.Identity
			creds    map[string]string
		)

		BeforeEach(func() {
			repoSpec = &runtime.Unstructured{
				Data: map[string]interface{}{
					"type": "oci",
				},
			}
			identity = runtime.Identity{"type": "ociRegistry", "hostname": "ghcr.io"}
			creds = map[string]string{"username": "user", "password": "pass"}
		})

		Context("reference and alias validation", func() {
			It("returns error when reference version is empty", func() {
				ref := compref.Ref{
					Repository: repoSpec,
					Component:  "github.com/acme/backend",
					Version:    "",
				}

				err := client.AddAlias(ctx, ref, "latest")

				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, konfcompref.ErrInvalidComponentReference)).To(BeTrue())
				Expect(err.Error()).To(ContainSubstring("invalid reference for AddAlias"))
			})

			It("returns error when alias is empty", func() {
				ref := compref.Ref{
					Repository: repoSpec,
					Component:  "github.com/acme/backend",
					Version:    "1.0.0",
				}

				err := client.AddAlias(ctx, ref, "")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("alias must not be empty"))
			})
		})

		Context("successful alias creation", func() {
			It("creates a new alias for a semantic version", func() {
				ref := compref.Ref{
					Repository: repoSpec,
					Component:  "github.com/acme/backend",
					Version:    "1.2.3",
				}

				providerMock.EXPECT().
					GetComponentVersionRepositoryCredentialConsumerIdentity(gomock.Any(), repoSpec).
					Return(identity, nil)
				resolverMock.EXPECT().
					Resolve(gomock.Any(), identity).
					Return(creds, nil)
				providerMock.EXPECT().
					GetComponentVersionRepository(gomock.Any(), repoSpec, creds).
					Return(repoMock, nil)
				repoMock.EXPECT().
					AddComponentVersionAlias(gomock.Any(), "github.com/acme/backend", "1.2.3", "latest").
					Return(nil)

				err := client.AddAlias(ctx, ref, "latest")

				Expect(err).ToNot(HaveOccurred())
			})

			It("creates an alias pointing to an existing alias", func() {
				ref := compref.Ref{
					Repository: repoSpec,
					Component:  "github.com/acme/backend",
					Version:    "latest", // Point to existing alias
				}

				providerMock.EXPECT().
					GetComponentVersionRepositoryCredentialConsumerIdentity(gomock.Any(), repoSpec).
					Return(identity, nil)
				resolverMock.EXPECT().
					Resolve(gomock.Any(), identity).
					Return(creds, nil)
				providerMock.EXPECT().
					GetComponentVersionRepository(gomock.Any(), repoSpec, creds).
					Return(repoMock, nil)
				repoMock.EXPECT().
					AddComponentVersionAlias(gomock.Any(), "github.com/acme/backend", "latest", "stable").
					Return(nil)

				err := client.AddAlias(ctx, ref, "stable")

				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("error handling", func() {
			It("returns ErrNotFound when component version doesn't exist", func() {
				ref := compref.Ref{
					Repository: repoSpec,
					Component:  "github.com/acme/backend",
					Version:    "99.99.99",
				}

				providerMock.EXPECT().
					GetComponentVersionRepositoryCredentialConsumerIdentity(gomock.Any(), repoSpec).
					Return(identity, nil)
				resolverMock.EXPECT().
					Resolve(gomock.Any(), identity).
					Return(creds, nil)
				providerMock.EXPECT().
					GetComponentVersionRepository(gomock.Any(), repoSpec, creds).
					Return(repoMock, nil)
				repoMock.EXPECT().
					AddComponentVersionAlias(gomock.Any(), "github.com/acme/backend", "99.99.99", "latest").
					Return(repository.ErrNotFound)

				err := client.AddAlias(ctx, ref, "latest")

				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, ErrNotFound)).To(BeTrue())
			})

			It("returns error when getting repository credential identity fails", func() {
				ref := compref.Ref{
					Repository: repoSpec,
					Component:  "github.com/acme/backend",
					Version:    "1.0.0",
				}

				providerMock.EXPECT().
					GetComponentVersionRepositoryCredentialConsumerIdentity(gomock.Any(), repoSpec).
					Return(nil, fmt.Errorf("invalid repo spec"))

				err := client.AddAlias(ctx, ref, "latest")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("getting repository for component reference"))
				Expect(err.Error()).To(ContainSubstring("invalid repo spec"))
			})

			It("returns error when credential resolution fails with non-NotFound error", func() {
				ref := compref.Ref{
					Repository: repoSpec,
					Component:  "github.com/acme/backend",
					Version:    "1.0.0",
				}

				providerMock.EXPECT().
					GetComponentVersionRepositoryCredentialConsumerIdentity(gomock.Any(), repoSpec).
					Return(identity, nil)
				resolverMock.EXPECT().
					Resolve(gomock.Any(), identity).
					Return(nil, fmt.Errorf("auth service unavailable"))

				err := client.AddAlias(ctx, ref, "latest")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("auth service unavailable"))
			})

			It("returns error when getting repository fails", func() {
				ref := compref.Ref{
					Repository: repoSpec,
					Component:  "github.com/acme/backend",
					Version:    "1.0.0",
				}

				providerMock.EXPECT().
					GetComponentVersionRepositoryCredentialConsumerIdentity(gomock.Any(), repoSpec).
					Return(identity, nil)
				resolverMock.EXPECT().
					Resolve(gomock.Any(), identity).
					Return(creds, nil)
				providerMock.EXPECT().
					GetComponentVersionRepository(gomock.Any(), repoSpec, creds).
					Return(nil, fmt.Errorf("repository unavailable"))

				err := client.AddAlias(ctx, ref, "latest")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("repository unavailable"))
			})

			It("returns error when AddComponentVersionAlias fails", func() {
				ref := compref.Ref{
					Repository: repoSpec,
					Component:  "github.com/acme/backend",
					Version:    "1.0.0",
				}

				providerMock.EXPECT().
					GetComponentVersionRepositoryCredentialConsumerIdentity(gomock.Any(), repoSpec).
					Return(identity, nil)
				resolverMock.EXPECT().
					Resolve(gomock.Any(), identity).
					Return(creds, nil)
				providerMock.EXPECT().
					GetComponentVersionRepository(gomock.Any(), repoSpec, creds).
					Return(repoMock, nil)
				repoMock.EXPECT().
					AddComponentVersionAlias(gomock.Any(), "github.com/acme/backend", "1.0.0", "latest").
					Return(fmt.Errorf("permission denied"))

				err := client.AddAlias(ctx, ref, "latest")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("adding alias latest"))
				Expect(err.Error()).To(ContainSubstring("permission denied"))
			})
		})
	})

	Describe("GetLocalResource", func() {
		var (
			repoSpec    *runtime.Unstructured
			identity    runtime.Identity
			creds       map[string]string
			ref         compref.Ref
			resIdentity runtime.Identity
		)

		BeforeEach(func() {
			repoSpec = &runtime.Unstructured{
				Data: map[string]interface{}{
					"type": "oci",
				},
			}
			identity = runtime.Identity{"type": "ociRegistry", "hostname": "ghcr.io"}
			creds = map[string]string{"username": "user", "password": "pass"}
			ref = compref.Ref{
				Repository: repoSpec,
				Component:  "github.com/acme/backend",
				Version:    "1.0.0",
			}
			resIdentity = runtime.Identity{"name": "my-manifest"}
		})

		Context("successful retrieval", func() {
			It("retrieves a local resource successfully", func() {
				expectedResource := &descruntime.Resource{
					ElementMeta: descruntime.ElementMeta{
						ObjectMeta: descruntime.ObjectMeta{
							Name: "my-manifest",
						},
					},
				}
				providerMock.EXPECT().
					GetComponentVersionRepositoryCredentialConsumerIdentity(gomock.Any(), repoSpec).
					Return(identity, nil)
				resolverMock.EXPECT().
					Resolve(gomock.Any(), identity).
					Return(creds, nil)
				providerMock.EXPECT().
					GetComponentVersionRepository(gomock.Any(), repoSpec, creds).
					Return(repoMock, nil)
				repoMock.EXPECT().
					GetLocalResource(gomock.Any(), ref.Component, ref.Version, resIdentity).
					Return(nil, expectedResource, nil)

				content, res, err := client.GetLocalResource(ctx, ref, resIdentity)

				Expect(err).ToNot(HaveOccurred())
				Expect(content).To(BeNil())
				Expect(res).To(Equal(expectedResource))
			})

			It("works with anonymous access when no credentials found", func() {
				expectedResource := &descruntime.Resource{
					ElementMeta: descruntime.ElementMeta{
						ObjectMeta: descruntime.ObjectMeta{
							Name: "my-manifest",
						},
					},
				}

				providerMock.EXPECT().
					GetComponentVersionRepositoryCredentialConsumerIdentity(gomock.Any(), repoSpec).
					Return(identity, nil)
				resolverMock.EXPECT().
					Resolve(gomock.Any(), identity).
					Return(nil, fmt.Errorf("credentials: %w", credentials.ErrNotFound))
				providerMock.EXPECT().
					GetComponentVersionRepository(gomock.Any(), repoSpec, gomock.Nil()).
					Return(repoMock, nil)
				repoMock.EXPECT().
					GetLocalResource(gomock.Any(), ref.Component, ref.Version, resIdentity).
					Return(nil, expectedResource, nil)

				content, res, err := client.GetLocalResource(ctx, ref, resIdentity)

				Expect(err).ToNot(HaveOccurred())
				Expect(content).To(BeNil())
				Expect(res).To(Equal(expectedResource))
			})
		})

		Context("error handling", func() {
			It("returns ErrInvalidComponentReference for invalid reference", func() {
				invalidRef := compref.Ref{
					Repository: repoSpec,
					Component:  "github.com/acme/backend",
					Version:    "", // empty version is invalid
				}

				content, res, err := client.GetLocalResource(ctx, invalidRef, resIdentity)

				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, ErrInvalidComponentReference)).To(BeTrue())
				Expect(content).To(BeNil())
				Expect(res).To(BeNil())
			})

			It("returns ErrNotFound when resource doesn't exist", func() {
				providerMock.EXPECT().
					GetComponentVersionRepositoryCredentialConsumerIdentity(gomock.Any(), repoSpec).
					Return(identity, nil)
				resolverMock.EXPECT().
					Resolve(gomock.Any(), identity).
					Return(creds, nil)
				providerMock.EXPECT().
					GetComponentVersionRepository(gomock.Any(), repoSpec, creds).
					Return(repoMock, nil)
				repoMock.EXPECT().
					GetLocalResource(gomock.Any(), ref.Component, ref.Version, resIdentity).
					Return(nil, nil, repository.ErrNotFound)

				content, res, err := client.GetLocalResource(ctx, ref, resIdentity)

				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, ErrNotFound)).To(BeTrue())
				Expect(content).To(BeNil())
				Expect(res).To(BeNil())
			})

			It("returns error when repository access fails", func() {
				providerMock.EXPECT().
					GetComponentVersionRepositoryCredentialConsumerIdentity(gomock.Any(), repoSpec).
					Return(identity, nil)
				resolverMock.EXPECT().
					Resolve(gomock.Any(), identity).
					Return(creds, nil)
				providerMock.EXPECT().
					GetComponentVersionRepository(gomock.Any(), repoSpec, creds).
					Return(nil, fmt.Errorf("connection failed"))

				content, res, err := client.GetLocalResource(ctx, ref, resIdentity)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("connection failed"))
				Expect(content).To(BeNil())
				Expect(res).To(BeNil())
			})

			It("returns error when credential identity lookup fails", func() {
				providerMock.EXPECT().
					GetComponentVersionRepositoryCredentialConsumerIdentity(gomock.Any(), repoSpec).
					Return(nil, fmt.Errorf("invalid repo spec"))

				content, res, err := client.GetLocalResource(ctx, ref, resIdentity)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("invalid repo spec"))
				Expect(content).To(BeNil())
				Expect(res).To(BeNil())
			})

			It("returns error when GetLocalResource fails", func() {
				providerMock.EXPECT().
					GetComponentVersionRepositoryCredentialConsumerIdentity(gomock.Any(), repoSpec).
					Return(identity, nil)
				resolverMock.EXPECT().
					Resolve(gomock.Any(), identity).
					Return(creds, nil)
				providerMock.EXPECT().
					GetComponentVersionRepository(gomock.Any(), repoSpec, creds).
					Return(repoMock, nil)
				repoMock.EXPECT().
					GetLocalResource(gomock.Any(), ref.Component, ref.Version, resIdentity).
					Return(nil, nil, fmt.Errorf("download failed"))

				content, res, err := client.GetLocalResource(ctx, ref, resIdentity)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("download failed"))
				Expect(content).To(BeNil())
				Expect(res).To(BeNil())
			})
		})
	})
})

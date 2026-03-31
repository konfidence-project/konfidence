package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/go-logr/logr"
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
		})

		Context("with latest version (empty version)", func() {
			It("retrieves the latest version successfully", func() {
				ref := compref.Ref{
					Repository: repoSpec,
					Component:  "github.com/acme/backend",
					Version:    "", // Request latest
				}
				versions := []string{"2.0.0", "1.5.0", "1.0.0"}
				expectedDesc := makeDescriptor("github.com/acme/backend", "2.0.0")

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
					ListComponentVersions(gomock.Any(), "github.com/acme/backend").
					Return(versions, nil)
				repoMock.EXPECT().
					GetComponentVersion(gomock.Any(), "github.com/acme/backend", "2.0.0").
					Return(&expectedDesc, nil)

				desc, err := client.Get(ctx, ref)

				Expect(err).ToNot(HaveOccurred())
				Expect(desc).To(Equal(expectedDesc))
			})

			It("returns ErrNotFound when component has no versions", func() {
				ref := compref.Ref{
					Repository: repoSpec,
					Component:  "github.com/acme/empty",
					Version:    "",
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
					ListComponentVersions(gomock.Any(), "github.com/acme/empty").
					Return([]string{}, nil)

				desc, err := client.Get(ctx, ref)

				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, ErrNotFound)).To(BeTrue())
				Expect(desc).To(Equal(descruntime.Descriptor{}))
			})

			It("returns ErrNotFound when repository doesn't exist", func() {
				ref := compref.Ref{
					Repository: repoSpec,
					Component:  "github.com/acme/missing",
					Version:    "",
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
					ListComponentVersions(gomock.Any(), "github.com/acme/missing").
					Return(nil, fmt.Errorf("repository name not known to registry"))

				desc, err := client.Get(ctx, ref)

				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, ErrNotFound)).To(BeTrue())
				Expect(desc).To(Equal(descruntime.Descriptor{}))
			})

			It("returns error when listing versions fails", func() {
				ref := compref.Ref{
					Repository: repoSpec,
					Component:  "github.com/acme/backend",
					Version:    "",
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
					ListComponentVersions(gomock.Any(), "github.com/acme/backend").
					Return(nil, fmt.Errorf("network error"))

				desc, err := client.Get(ctx, ref)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("network error"))
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
})

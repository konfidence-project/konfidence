package ocm

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ocmgenericspecv1 "ocm.software/open-component-model/bindings/go/configuration/generic/v1/spec"
	"ocm.software/open-component-model/bindings/go/credentials"
	ocmcompref "ocm.software/open-component-model/bindings/go/oci/compref"
	"ocm.software/open-component-model/bindings/go/plugin/manager"
	"ocm.software/open-component-model/bindings/go/repository"
	"ocm.software/open-component-model/bindings/go/repository/component/resolvers"
)

// mockAliasRepo extends mockRepo with AliasComponentVersionRepository support.
type mockAliasRepo struct {
	mockRepo
	addAliasErr    error
	addAliasCalled bool
}

func (m *mockAliasRepo) AddComponentVersionAlias(_ context.Context, _, _, _ string) error {
	m.addAliasCalled = true
	return m.addAliasErr
}

func (m *mockAliasRepo) RemoveComponentVersionAlias(_ context.Context, _, _ string) error {
	return nil
}

var _ = Describe("Alias", func() {
	var (
		ctx       context.Context
		ocmConfig ocmgenericspecv1.Config
		props     AliasProperties
	)

	BeforeEach(func() {
		ctx = context.Background()
		ocmConfig = ocmgenericspecv1.Config{}
		props = AliasProperties{
			ComponentVersion: "registry.example.com//github.com/test/component:1.0.0",
			Alias:            "edge",
		}

		ocmGetPluginManager = func(_ context.Context) (*manager.PluginManager, error) {
			return &manager.PluginManager{}, nil
		}
		ocmGetCredentialGraph = func(_ context.Context, _ *manager.PluginManager, _ *ocmgenericspecv1.Config) (credentials.Resolver, error) {
			return &mockCredentialsResolver{}, nil
		}
		ocmParseComponentReference = func(_ string, _ ...ocmcompref.Option) (*ocmcompref.Ref, error) {
			ref, _ := ocmcompref.Parse("registry.example.com//github.com/test/component:1.0.0")
			return ref, nil
		}
		ocmNewComponentRepositoryResolver = func(_ context.Context, _ repository.ComponentVersionRepositoryProvider,
			_ credentials.Resolver, _ ...RepositoryResolverOption) (resolvers.ComponentVersionRepositoryResolver, error) {
			return &mockRepoResolver{repo: &mockAliasRepo{}}, nil
		}
	})

	Context("happy path", func() {
		It("creates the alias successfully", func() {
			repo := &mockAliasRepo{}
			ocmNewComponentRepositoryResolver = func(_ context.Context, _ repository.ComponentVersionRepositoryProvider,
				_ credentials.Resolver, _ ...RepositoryResolverOption) (resolvers.ComponentVersionRepositoryResolver, error) {
				return &mockRepoResolver{repo: repo}, nil
			}
			Expect(Alias(ctx, props, &ocmConfig)).To(Succeed())
			Expect(repo.addAliasCalled).To(BeTrue())
		})
	})

	Context("with failing OCM library methods", func() {
		It("returns an error when getting the plugin manager fails", func() {
			ocmGetPluginManager = func(_ context.Context) (*manager.PluginManager, error) {
				return nil, fmt.Errorf("plugin manager unavailable")
			}
			Expect(Alias(ctx, props, &ocmConfig)).To(MatchError(ContainSubstring("failed to get plugin manager")))
		})

		It("returns an error when getting the credential graph fails", func() {
			ocmGetCredentialGraph = func(_ context.Context, _ *manager.PluginManager, _ *ocmgenericspecv1.Config) (credentials.Resolver, error) {
				return nil, fmt.Errorf("credential graph unavailable")
			}
			Expect(Alias(ctx, props, &ocmConfig)).To(MatchError(ContainSubstring("failed to get credential graph")))
		})

		It("returns an error when parsing the component reference fails", func() {
			ocmParseComponentReference = func(_ string, _ ...ocmcompref.Option) (*ocmcompref.Ref, error) {
				return nil, fmt.Errorf("invalid reference")
			}
			Expect(Alias(ctx, props, &ocmConfig)).To(MatchError(ContainSubstring("failed to parse component version")))
		})

		It("returns an error when creating the repository resolver fails", func() {
			ocmNewComponentRepositoryResolver = func(_ context.Context, _ repository.ComponentVersionRepositoryProvider,
				_ credentials.Resolver, _ ...RepositoryResolverOption) (resolvers.ComponentVersionRepositoryResolver, error) {
				return nil, fmt.Errorf("resolver unavailable")
			}
			Expect(Alias(ctx, props, &ocmConfig)).To(MatchError(ContainSubstring("failed to initialize ocm repository")))
		})

		It("returns an error when the resolver cannot find a repository for the component", func() {
			ocmNewComponentRepositoryResolver = func(_ context.Context, _ repository.ComponentVersionRepositoryProvider,
				_ credentials.Resolver, _ ...RepositoryResolverOption) (resolvers.ComponentVersionRepositoryResolver, error) {
				return &mockRepoResolver{err: fmt.Errorf("component not found")}, nil
			}
			Expect(Alias(ctx, props, &ocmConfig)).To(MatchError(ContainSubstring("failed to access ocm repository")))
		})

		It("returns an error when the repository does not support aliasing", func() {
			ocmNewComponentRepositoryResolver = func(_ context.Context, _ repository.ComponentVersionRepositoryProvider,
				_ credentials.Resolver, _ ...RepositoryResolverOption) (resolvers.ComponentVersionRepositoryResolver, error) {
				return &mockRepoResolver{repo: &mockRepo{}}, nil
			}
			Expect(Alias(ctx, props, &ocmConfig)).To(MatchError(ContainSubstring("repository does not support aliasing")))
		})

		It("returns an error when AddComponentVersionAlias fails", func() {
			ocmNewComponentRepositoryResolver = func(_ context.Context, _ repository.ComponentVersionRepositoryProvider,
				_ credentials.Resolver, _ ...RepositoryResolverOption) (resolvers.ComponentVersionRepositoryResolver, error) {
				return &mockRepoResolver{repo: &mockAliasRepo{addAliasErr: fmt.Errorf("alias error")}}, nil
			}
			Expect(Alias(ctx, props, &ocmConfig)).To(MatchError(ContainSubstring("alias error")))
		})
	})
})

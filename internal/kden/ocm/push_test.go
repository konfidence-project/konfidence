package ocm

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"ocm.software/open-component-model/bindings/go/blob"
	ocmgenericspecv1 "ocm.software/open-component-model/bindings/go/configuration/generic/v1/spec"
	ocmconstructor "ocm.software/open-component-model/bindings/go/constructor"
	ocmconstructorruntime "ocm.software/open-component-model/bindings/go/constructor/runtime"
	ocmconstructorspecv1 "ocm.software/open-component-model/bindings/go/constructor/spec/v1"
	"ocm.software/open-component-model/bindings/go/credentials"
	syncdag "ocm.software/open-component-model/bindings/go/dag/sync"
	descriptorruntime "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	"ocm.software/open-component-model/bindings/go/plugin/manager"
	"ocm.software/open-component-model/bindings/go/repository"
	"ocm.software/open-component-model/bindings/go/repository/component/resolvers"
	"ocm.software/open-component-model/bindings/go/runtime"
)

type mockConstructor struct{}

func (m *mockConstructor) Construct(_ context.Context) error { return nil }
func (m *mockConstructor) GetGraph() *syncdag.SyncedDirectedAcyclicGraph[string] {
	return nil
}

// stubCredentialResolver is a no-op credentials.Resolver used to assert that a
// credential graph is wired through to the component repository resolver.
type stubCredentialResolver struct{}

func (stubCredentialResolver) Resolve(_ context.Context, _ runtime.Identity) (runtime.Typed, error) {
	return nil, nil
}

type failingConstructor struct{}

func (f *failingConstructor) Construct(_ context.Context) error {
	return fmt.Errorf("construct failed")
}
func (f *failingConstructor) GetGraph() *syncdag.SyncedDirectedAcyclicGraph[string] {
	return nil
}

type mockTargetRepository struct{}

func (m *mockTargetRepository) AddLocalResource(_ context.Context, _, _ string,
	res *descriptorruntime.Resource, _ blob.ReadOnlyBlob) (*descriptorruntime.Resource, error) {
	return res, nil
}
func (m *mockTargetRepository) AddLocalSource(_ context.Context, _, _ string,
	src *descriptorruntime.Source, _ blob.ReadOnlyBlob) (*descriptorruntime.Source, error) {
	return src, nil
}
func (m *mockTargetRepository) AddComponentVersion(_ context.Context, _ *descriptorruntime.Descriptor) error {
	return nil
}
func (m *mockTargetRepository) GetComponentVersion(_ context.Context, _, _ string) (*descriptorruntime.Descriptor, error) {
	return &descriptorruntime.Descriptor{}, nil
}
func (m *mockTargetRepository) AddComponentVersionAlias(_ context.Context, _, _, _ string) error {
	return nil
}
func (m *mockTargetRepository) RemoveComponentVersionAlias(_ context.Context, _, _ string) error {
	return nil
}

type failingTargetRepository struct{}

func (m *failingTargetRepository) AddLocalResource(_ context.Context, _, _ string,
	res *descriptorruntime.Resource, _ blob.ReadOnlyBlob) (*descriptorruntime.Resource, error) {
	return res, nil
}
func (m *failingTargetRepository) AddLocalSource(_ context.Context, _, _ string,
	src *descriptorruntime.Source, _ blob.ReadOnlyBlob) (*descriptorruntime.Source, error) {
	return src, nil
}
func (m *failingTargetRepository) AddComponentVersion(_ context.Context, _ *descriptorruntime.Descriptor) error {
	return nil
}
func (m *failingTargetRepository) GetComponentVersion(_ context.Context, _, _ string) (*descriptorruntime.Descriptor, error) {
	return &descriptorruntime.Descriptor{}, nil
}
func (m *failingTargetRepository) AddComponentVersionAlias(_ context.Context, _, _, _ string) error {
	return fmt.Errorf("alias failed")
}
func (m *failingTargetRepository) RemoveComponentVersionAlias(_ context.Context, _, _ string) error {
	return fmt.Errorf("alias failed")
}

// existingVersionsRepository reports only the listed "name:version" pairs as
// present, like a target repository that holds some components already.
type existingVersionsRepository struct {
	mockTargetRepository
	existing map[string]bool
	lookups  []string
}

func (r *existingVersionsRepository) GetComponentVersion(_ context.Context, name, version string) (*descriptorruntime.Descriptor, error) {
	r.lookups = append(r.lookups, name+":"+version)
	if !r.existing[name+":"+version] {
		return nil, repository.ErrNotFound
	}
	return &descriptorruntime.Descriptor{}, nil
}

func newComponent(name, version string, refs ...ocmconstructorruntime.Reference) ocmconstructorruntime.Component {
	c := ocmconstructorruntime.Component{References: refs}
	c.Name, c.Version = name, version
	return c
}

func newReference(component, version string) ocmconstructorruntime.Reference {
	ref := ocmconstructorruntime.Reference{Component: component}
	ref.Name, ref.Version = component, version
	return ref
}

var _ = BeforeEach(func() {
	ocmGetPluginManager = func(_ context.Context, _ *ocmgenericspecv1.Config) (*manager.PluginManager, error) {
		return &manager.PluginManager{}, nil
	}
	ocmGetRepositorySpec = func(_ string) (runtime.Typed, error) {
		return nil, nil
	}
	ocmGetComponentRepositoryResolver = func(
		_ context.Context,
		_ repository.ComponentVersionRepositoryProvider,
		_ credentials.Resolver,
		_ ...RepositoryResolverOption,
	) (resolvers.ComponentVersionRepositoryResolver, error) {
		return nil, nil
	}
	ocmGetCredentialGraph = func(_ context.Context, _ *manager.PluginManager, _ *ocmgenericspecv1.Config) (credentials.Resolver, error) {
		return nil, nil
	}
	ocmConvertToRuntimeConstructor = func(_ *ocmconstructorspecv1.ComponentConstructor) *ocmconstructorruntime.ComponentConstructor {
		return &ocmconstructorruntime.ComponentConstructor{}
	}
	ocmCreateConstructor = func(_ *ocmconstructorruntime.ComponentConstructor, _ ocmconstructor.Options) ocmconstructor.Constructor {
		return &mockConstructor{}
	}
})

var _ = Describe("GetOcmConstructorProvider", func() {

	var ocmConfig ocmgenericspecv1.Config
	var ctx context.Context
	var registry = "https://some-registry.com"

	var _ = BeforeEach(func() {
		ocmConfig = ocmgenericspecv1.Config{}
		ctx = context.Background()
	})

	Context("when the component configuration is valid", func() {
		It("creates the constructor provider successfully", func() {
			_, err := GetOcmConstructorProvider(&ocmConfig, ctx, registry)
			Expect(err).ToNot(HaveOccurred())
		})

		It("passes the credential graph to the component repository resolver", func() {
			// Regression: the resolver must receive the credential graph so
			// resolving external component references is authenticated (not nil).
			origGraph := ocmGetCredentialGraph
			origResolver := ocmGetComponentRepositoryResolver
			defer func() {
				ocmGetCredentialGraph = origGraph
				ocmGetComponentRepositoryResolver = origResolver
			}()

			sentinel := stubCredentialResolver{}
			ocmGetCredentialGraph = func(_ context.Context, _ *manager.PluginManager,
				_ *ocmgenericspecv1.Config) (credentials.Resolver, error) {
				return sentinel, nil
			}
			var passed credentials.Resolver
			ocmGetComponentRepositoryResolver = func(_ context.Context,
				_ repository.ComponentVersionRepositoryProvider, graph credentials.Resolver,
				_ ...RepositoryResolverOption) (resolvers.ComponentVersionRepositoryResolver, error) {
				passed = graph
				return nil, nil
			}

			_, err := GetOcmConstructorProvider(&ocmConfig, ctx, registry)
			Expect(err).ToNot(HaveOccurred())
			Expect(passed).To(Equal(sentinel))
		})
	})

	Context("with failing OCM Library methods", func() {
		It("returns an error when getting the plugin manager throws an error", func() {
			ocmGetPluginManager = func(_ context.Context, _ *ocmgenericspecv1.Config) (*manager.PluginManager, error) {
				return nil, fmt.Errorf("ocm library fails")
			}
			_, err := GetOcmConstructorProvider(&ocmConfig, ctx, registry)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to load plugin manager"))
		})

		It("returns an error when getting the repository spec throws an error", func() {
			ocmGetRepositorySpec = func(_ string) (runtime.Typed, error) {
				return nil, fmt.Errorf("ocm library fails")
			}
			_, err := GetOcmConstructorProvider(&ocmConfig, ctx, registry)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to load repository spec"))
		})

		It("returns an error when getting the component repository resolver throws an error", func() {
			ocmGetComponentRepositoryResolver = func(
				_ context.Context,
				_ repository.ComponentVersionRepositoryProvider,
				_ credentials.Resolver,
				_ ...RepositoryResolverOption,
			) (resolvers.ComponentVersionRepositoryResolver, error) {
				return nil, fmt.Errorf("ocm library fails")
			}
			_, err := GetOcmConstructorProvider(&ocmConfig, ctx, registry)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get repository resolver"))
		})

		It("returns an error when getting the credential graph throws an error", func() {
			ocmGetCredentialGraph = func(
				_ context.Context,
				_ *manager.PluginManager,
				_ *ocmgenericspecv1.Config,
			) (credentials.Resolver, error) {
				return nil, fmt.Errorf("ocm library fails")
			}
			_, err := GetOcmConstructorProvider(&ocmConfig, ctx, registry)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get credential graph"))
		})
	})
})

var _ = Describe("PushComponentConstructor", func() {

	var ctx context.Context
	var constructor ocmconstructorspecv1.ComponentConstructor
	var constructorOptionsProvider ConstructorProvider

	var _ = BeforeEach(func() {
		ctx = context.Background()
		constructor = ocmconstructorspecv1.ComponentConstructor{}
	})

	Context("when the component constructor is valid", func() {
		It("pushes the component constructor successfully", func() {
			err := PushComponentConstructor(ctx, &constructorOptionsProvider, &constructor)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("with failing OCM Library methods", func() {
		It("returns an error when the conversion to a runtime constructor fails", func() {
			ocmConvertToRuntimeConstructor = func(_ *ocmconstructorspecv1.ComponentConstructor) *ocmconstructorruntime.ComponentConstructor {
				return nil
			}
			err := PushComponentConstructor(ctx, &constructorOptionsProvider, &constructor)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to create runtime constructor"))
		})

		It("returns an error when creating the constructor throws an error", func() {
			ocmCreateConstructor = func(_ *ocmconstructorruntime.ComponentConstructor, _ ocmconstructor.Options) ocmconstructor.Constructor {
				return &failingConstructor{}
			}
			err := PushComponentConstructor(ctx, &constructorOptionsProvider, &constructor)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("construct failed"))
		})
	})

	// Regression for #414: copying referenced components that already live in
	// the target repository fails intermittently with "context canceled".
	Context("choosing the copy policy for referenced components", func() {
		var target *existingVersionsRepository
		var targetErr error
		var opts ocmconstructor.Options
		// The OCM constant is untyped, so convert it for Equal.
		skip := ocmconstructor.ExternalComponentVersionCopyPolicy(ocmconstructor.ExternalComponentVersionCopyPolicySkip)

		push := func(components ...ocmconstructorruntime.Component) {
			ocmConvertToRuntimeConstructor = func(_ *ocmconstructorspecv1.ComponentConstructor) *ocmconstructorruntime.ComponentConstructor {
				return &ocmconstructorruntime.ComponentConstructor{Components: components}
			}
			Expect(PushComponentConstructor(ctx, &constructorOptionsProvider, &constructor)).To(Succeed())
		}

		BeforeEach(func() {
			target = &existingVersionsRepository{existing: map[string]bool{"example.com/candidates:1.0.0": true}}
			targetErr = nil
			ocmGetTargetRepository = func(_ context.Context, _ *ocmconstructorruntime.Component,
				_ *ConstructorProvider) (ocmconstructor.TargetRepository, error) {
				return target, targetErr
			}
			ocmCreateConstructor = func(_ *ocmconstructorruntime.ComponentConstructor, o ocmconstructor.Options) ocmconstructor.Constructor {
				opts = o
				return &mockConstructor{}
			}
		})

		It("skips the copy when every referenced component exists in the target repository", func() {
			push(newComponent("example.com/vector", "2.0.0", newReference("example.com/candidates", "1.0.0")))
			Expect(opts.ExternalComponentVersionCopyPolicy).To(Equal(skip))
		})

		It("copies when a referenced component is missing from the target repository", func() {
			push(newComponent("example.com/vector", "2.0.0",
				newReference("example.com/candidates", "1.0.0"),
				newReference("example.com/interviews", "1.0.0")))
			Expect(opts.ExternalComponentVersionCopyPolicy).To(Equal(ocmconstructor.ExternalComponentVersionCopyPolicyCopyOrFail))
		})

		It("copies when the target repository cannot be opened", func() {
			targetErr = fmt.Errorf("no repository")
			push(newComponent("example.com/vector", "2.0.0", newReference("example.com/candidates", "1.0.0")))
			Expect(opts.ExternalComponentVersionCopyPolicy).To(Equal(ocmconstructor.ExternalComponentVersionCopyPolicyCopyOrFail))
		})

		It("ignores references to components built by the same constructor", func() {
			push(
				newComponent("example.com/vector", "2.0.0",
					newReference("example.com/candidates", "1.0.0"),
					newReference("example.com/config", "2.0.0")),
				newComponent("example.com/config", "2.0.0"))
			Expect(opts.ExternalComponentVersionCopyPolicy).To(Equal(skip))
			Expect(target.lookups).To(Equal([]string{"example.com/candidates:1.0.0"}))
		})

		It("keeps the default and does not query the target repository without references", func() {
			push(newComponent("example.com/candidates", "1.0.0"))
			Expect(opts.ExternalComponentVersionCopyPolicy).To(Equal(ocmconstructor.ExternalComponentVersionCopyPolicyCopyOrFail))
			Expect(target.lookups).To(BeEmpty())
		})
	})
})

var _ = Describe("AddComponentVersionAlias", func() {

	var ctx context.Context
	var component = "componentName"
	var versionOrAlias = "1.0.0"
	var alias = "latest"
	var constructorOptionsProvider ConstructorProvider

	var _ = BeforeEach(func() {
		ctx = context.Background()
	})

	Context("when the aliasing is valid", func() {

		It("creates the constructor provider successfully", func() {
			ocmGetTargetRepository = func(
				_ context.Context,
				_ *ocmconstructorruntime.Component,
				_ *ConstructorProvider,
			) (ocmconstructor.TargetRepository, error) {
				return &mockTargetRepository{}, nil
			}
			err := AddComponentVersionAlias(ctx, component, versionOrAlias, alias, &constructorOptionsProvider)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("with failing OCM Library methods", func() {
		It("returns an error when getting the target repository throws an error", func() {
			ocmGetTargetRepository = func(
				_ context.Context,
				_ *ocmconstructorruntime.Component,
				_ *ConstructorProvider,
			) (ocmconstructor.TargetRepository, error) {
				return nil, fmt.Errorf("failed to get target repository")
			}

			err := AddComponentVersionAlias(ctx, component, versionOrAlias, alias, &constructorOptionsProvider)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get target repository"))
		})
		It("returns an error when adding the component version alias throws an error", func() {
			ocmGetTargetRepository = func(
				_ context.Context,
				_ *ocmconstructorruntime.Component,
				_ *ConstructorProvider,
			) (ocmconstructor.TargetRepository, error) {
				return &failingTargetRepository{}, nil
			}

			err := AddComponentVersionAlias(ctx, component, versionOrAlias, alias, &constructorOptionsProvider)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("alias failed"))
		})
	})
})

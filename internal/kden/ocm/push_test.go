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

var _ = BeforeEach(func() {
	ocmGetPluginManager = func(_ context.Context, registry string) (*manager.PluginManager, error) {
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
	})

	Context("with failing OCM Library methods", func() {
		It("returns an error when getting the plugin manager throws an error", func() {
			ocmGetPluginManager = func(_ context.Context, registry string) (*manager.PluginManager, error) {
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

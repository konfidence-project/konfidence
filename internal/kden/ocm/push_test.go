package ocm

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ocmgenericspecv1 "ocm.software/open-component-model/bindings/go/configuration/generic/v1/spec"
	ocmconstructor "ocm.software/open-component-model/bindings/go/constructor"
	ocmconstructorruntime "ocm.software/open-component-model/bindings/go/constructor/runtime"
	ocmconstructorspecv1 "ocm.software/open-component-model/bindings/go/constructor/spec/v1"
	"ocm.software/open-component-model/bindings/go/credentials"
	syncdag "ocm.software/open-component-model/bindings/go/dag/sync"
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

var _ = BeforeEach(func() {

	OcmMethodsImpl = OcmMethods{
		GetPluginManager: func(_ context.Context) (*manager.PluginManager, error) {
			return &manager.PluginManager{}, nil
		},
		GetRepositorySpec: func(_ string) (runtime.Typed, error) {
			return nil, nil
		},
		GetComponentRepositoryResolver: func(
			_ context.Context,
			_ repository.ComponentVersionRepositoryProvider,
			_ credentials.Resolver,
			_ ...RepositoryResolverOption,
		) (resolvers.ComponentVersionRepositoryResolver, error) {
			return nil, nil
		},
		GetCredentialGraph: func(_ *manager.PluginManager, _ context.Context, _ *ocmgenericspecv1.Config) (credentials.Resolver, error) {
			return nil, nil
		},
		ConvertToRuntimeConstructor: func(_ *ocmconstructorspecv1.ComponentConstructor) *ocmconstructorruntime.ComponentConstructor {
			return &ocmconstructorruntime.ComponentConstructor{}
		},
		CreateConstructor: func(_ *ocmconstructorruntime.ComponentConstructor, _ ocmconstructor.Options) ocmconstructor.Constructor {
			return &mockConstructor{}
		},
	}
})

var _ = Describe("PushComponentConstructor", func() {

	var ocmConfig ocmgenericspecv1.Config
	var ctx context.Context
	var registry = "https://some-registry.com"
	var constructor ocmconstructorspecv1.ComponentConstructor

	var _ = BeforeEach(func() {
		ocmConfig = ocmgenericspecv1.Config{}
		ctx = context.Background()
		constructor = ocmconstructorspecv1.ComponentConstructor{}
	})

	Context("when the component constructor is valid", func() {
		It("pushes the component constructor successfully", func() {
			err := PushComponentConstructor(&ocmConfig, ctx, registry, &constructor)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("with failing OCM Library methods", func() {
		It("returns an error when getting the plugin manager throws an error", func() {
			OcmMethodsImpl.GetPluginManager = func(_ context.Context) (*manager.PluginManager, error) {
				return nil, fmt.Errorf("ocm library fails")
			}
			err := PushComponentConstructor(&ocmConfig, ctx, registry, &constructor)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to load plugin manager"))
		})

		It("returns an error when getting the repository spec throws an error", func() {
			OcmMethodsImpl.GetRepositorySpec = func(_ string) (runtime.Typed, error) {
				return nil, fmt.Errorf("ocm library fails")
			}
			err := PushComponentConstructor(&ocmConfig, ctx, registry, &constructor)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to load repository spec"))
		})

		It("returns an error when getting the component repository resolver throws an error", func() {
			OcmMethodsImpl.GetComponentRepositoryResolver = func(
				_ context.Context,
				_ repository.ComponentVersionRepositoryProvider,
				_ credentials.Resolver,
				_ ...RepositoryResolverOption,
			) (resolvers.ComponentVersionRepositoryResolver, error) {
				return nil, fmt.Errorf("ocm library fails")
			}
			err := PushComponentConstructor(&ocmConfig, ctx, registry, &constructor)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get repository resolver"))
		})

		It("returns an error when getting the credential graph throws an error", func() {
			OcmMethodsImpl.GetCredentialGraph = func(_ *manager.PluginManager, _ context.Context, _ *ocmgenericspecv1.Config) (credentials.Resolver, error) {
				return nil, fmt.Errorf("ocm library fails")
			}
			err := PushComponentConstructor(&ocmConfig, ctx, registry, &constructor)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get credential graph"))
		})

		It("returns an error when the conversion to a runtime constructor fails", func() {
			OcmMethodsImpl.ConvertToRuntimeConstructor = func(_ *ocmconstructorspecv1.ComponentConstructor) *ocmconstructorruntime.ComponentConstructor {
				return nil
			}
			err := PushComponentConstructor(&ocmConfig, ctx, registry, &constructor)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to create runtime constructor"))
		})

		It("returns an error when getting the repository spec throws an error", func() {
			OcmMethodsImpl.CreateConstructor = func(_ *ocmconstructorruntime.ComponentConstructor, _ ocmconstructor.Options) ocmconstructor.Constructor {
				return &failingConstructor{}
			}
			err := PushComponentConstructor(&ocmConfig, ctx, registry, &constructor)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("construct failed"))
		})
	})
})

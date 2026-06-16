package ocm

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"ocm.software/open-component-model/bindings/go/plugin/manager"
	"ocm.software/open-component-model/bindings/go/plugin/manager/registries/componentversionrepository"
	"ocm.software/open-component-model/bindings/go/plugin/manager/registries/credentialrepository"
	"ocm.software/open-component-model/bindings/go/plugin/manager/registries/input"
	"ocm.software/open-component-model/bindings/go/plugin/manager/registries/signinghandler"
	ocmruntime "ocm.software/open-component-model/bindings/go/runtime"
)

var (
	errComponentVersionRepo = errors.New("component version repo error")
	errFileInput            = errors.New("file input error")
	errDirInput             = errors.New("dir input error")
	errSourceFileInput      = errors.New("source file input error")
	errSourceDirInput       = errors.New("source dir input error")
	errCredentialRepo       = errors.New("credential repo error")
	errSigningHandler       = errors.New("signing handler error")
)

type mockRegistrar struct {
	failComponentVersionRepo bool
	failFileInput            bool
	failDirInput             bool
	failSourceFileInput      bool
	failSourceDirInput       bool
	failCredentialRepo       bool
	failSigningHandler       bool
	resourceInputCallCount   int
	sourceInputCallCount     int
}

func (m *mockRegistrar) registerComponentVersionRepositoryPlugin(_ componentversionrepository.BuiltinComponentVersionRepositoryProvider) error {
	if m.failComponentVersionRepo {
		return errComponentVersionRepo
	}
	return nil
}

func (m *mockRegistrar) registerResourceInputPlugin(_ input.BuiltinResourceInputMethod) error {
	m.resourceInputCallCount++
	if m.failFileInput && m.resourceInputCallCount == 1 {
		return errFileInput
	}
	if m.failDirInput && m.resourceInputCallCount == 2 {
		return errDirInput
	}
	return nil
}

func (m *mockRegistrar) registerSourceInputPlugin(_ input.BuiltinSourceInputMethod) error {
	m.sourceInputCallCount++
	if m.failSourceFileInput && m.sourceInputCallCount == 1 {
		return errSourceFileInput
	}
	if m.failSourceDirInput && m.sourceInputCallCount == 2 {
		return errSourceDirInput
	}
	return nil
}

func (m *mockRegistrar) registerCredentialRepositoryPlugin(_ credentialrepository.BuiltinCredentialRepositoryPlugin, _ []ocmruntime.Type) error {
	if m.failCredentialRepo {
		return errCredentialRepo
	}
	return nil
}

func (m *mockRegistrar) registerSigningHandler(_ signinghandler.BuiltinSigningHandler) error {
	if m.failSigningHandler {
		return errSigningHandler
	}
	return nil
}

var _ = Describe("GetPluginManager", func() {

	Context("with a valid context", func() {

		It("should return a non-nil plugin manager", func() {
			pluginManager, err := GetPluginManager(context.Background())

			Expect(err).ToNot(HaveOccurred())
			Expect(pluginManager).ToNot(BeNil())
		})

		It("should register the component version repository plugin", func() {
			pluginManager, err := GetPluginManager(context.Background())

			Expect(err).ToNot(HaveOccurred())
			Expect(pluginManager.ComponentVersionRepositoryRegistry).ToNot(BeNil())
		})

		It("should register the input plugins", func() {
			pluginManager, err := GetPluginManager(context.Background())

			Expect(err).ToNot(HaveOccurred())
			Expect(pluginManager.InputRegistry).ToNot(BeNil())
		})

		It("should register the credential repository plugin", func() {
			pluginManager, err := GetPluginManager(context.Background())

			Expect(err).ToNot(HaveOccurred())
			Expect(pluginManager.CredentialRepositoryRegistry).ToNot(BeNil())
		})

		It("should register the signing plugin", func() {
			pluginManager, err := GetPluginManager(context.Background())

			Expect(err).ToNot(HaveOccurred())
			Expect(pluginManager.SigningRegistry).ToNot(BeNil())
		})
	})

	Context("with a cancelled context", func() {

		It("should still return a plugin manager", func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			pluginManager, err := GetPluginManager(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(pluginManager).ToNot(BeNil())
		})
	})

	Context("when a registration step fails", func() {

		It("should return error when component version repository registration fails", func() {
			pm := manager.NewPluginManager(context.Background())
			_, err := setupPluginManager(context.Background(), pm, &mockRegistrar{failComponentVersionRepo: true})

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to register internal component version repository plugin"))
			Expect(errors.Is(err, errComponentVersionRepo)).To(BeTrue())
		})

		It("should return error when file resource input plugin registration fails", func() {
			pm := manager.NewPluginManager(context.Background())
			_, err := setupPluginManager(context.Background(), pm, &mockRegistrar{failFileInput: true})

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to register file input plugin"))
			Expect(errors.Is(err, errFileInput)).To(BeTrue())
		})

		It("should return error when dir resource input plugin registration fails", func() {
			pm := manager.NewPluginManager(context.Background())
			_, err := setupPluginManager(context.Background(), pm, &mockRegistrar{failDirInput: true})

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to register dir input plugin"))
			Expect(errors.Is(err, errDirInput)).To(BeTrue())
		})

		It("should return error when file source input plugin registration fails", func() {
			pm := manager.NewPluginManager(context.Background())
			_, err := setupPluginManager(context.Background(), pm, &mockRegistrar{failSourceFileInput: true})

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to register file input plugin"))
			Expect(errors.Is(err, errSourceFileInput)).To(BeTrue())
		})

		It("should return error when dir source input plugin registration fails", func() {
			pm := manager.NewPluginManager(context.Background())
			_, err := setupPluginManager(context.Background(), pm, &mockRegistrar{failSourceDirInput: true})

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to register dir input plugin"))
			Expect(errors.Is(err, errSourceDirInput)).To(BeTrue())
		})

		It("should return error when credential repository registration fails", func() {
			pm := manager.NewPluginManager(context.Background())
			_, err := setupPluginManager(context.Background(), pm, &mockRegistrar{failCredentialRepo: true})

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to register credential repository plugin"))
			Expect(errors.Is(err, errCredentialRepo)).To(BeTrue())
		})

		It("should return error when signing handler registration fails", func() {
			pm := manager.NewPluginManager(context.Background())
			_, err := setupPluginManager(context.Background(), pm, &mockRegistrar{failSigningHandler: true})

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to register internal signing plugin"))
			Expect(errors.Is(err, errSigningHandler)).To(BeTrue())
		})
	})
})

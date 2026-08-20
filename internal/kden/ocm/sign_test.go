package ocm

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
	"ocm.software/open-component-model/bindings/go/credentials"
	ocmsigningv1alpha1 "ocm.software/open-component-model/bindings/go/rsa/signing/v1alpha1"

	ocmblob "ocm.software/open-component-model/bindings/go/blob"
	ocmgenericspecv1 "ocm.software/open-component-model/bindings/go/configuration/generic/v1/spec"
	ocmdescriptorruntime "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	ocmcompref "ocm.software/open-component-model/bindings/go/oci/compref"
	ocmrepositoryctfv1 "ocm.software/open-component-model/bindings/go/oci/spec/repository/v1/ctf"
	"ocm.software/open-component-model/bindings/go/plugin/manager"
	ocmsigninghandler "ocm.software/open-component-model/bindings/go/plugin/manager/registries/signinghandler"
	"ocm.software/open-component-model/bindings/go/repository"
	"ocm.software/open-component-model/bindings/go/repository/component/resolvers"
	ocmruntime "ocm.software/open-component-model/bindings/go/runtime"
	ocmsigning "ocm.software/open-component-model/bindings/go/signing"
)

// mockRepoResolver implements resolvers.ComponentVersionRepositoryResolver.
type mockRepoResolver struct {
	repo repository.ComponentVersionRepository
	err  error
}

func (m *mockRepoResolver) GetComponentVersionRepositoryForComponent(_ context.Context, _, _ string) (repository.ComponentVersionRepository, error) {
	return m.repo, m.err
}
func (m *mockRepoResolver) GetComponentVersionRepositoryForSpecification(_ context.Context, _ ocmruntime.Typed) (repository.ComponentVersionRepository, error) {
	return m.repo, m.err
}
func (m *mockRepoResolver) GetRepositorySpecificationForComponent(_ context.Context, _, _ string) (ocmruntime.Typed, error) {
	return nil, nil
}

// mockRepo implements repository.ComponentVersionRepository.
type mockRepo struct {
	cv        *ocmdescriptorruntime.Descriptor
	cvErr     error
	addErr    error
	addCalled bool
}

func (m *mockRepo) GetComponentVersion(_ context.Context, _, _ string) (*ocmdescriptorruntime.Descriptor, error) {
	return m.cv, m.cvErr
}
func (m *mockRepo) AddComponentVersion(_ context.Context, _ *ocmdescriptorruntime.Descriptor) error {
	m.addCalled = true
	return m.addErr
}
func (m *mockRepo) ListComponentVersions(_ context.Context, _ string) ([]string, error) {
	return nil, nil
}
func (m *mockRepo) AddLocalResource(_ context.Context, _, _ string, _ *ocmdescriptorruntime.Resource,
	_ ocmblob.ReadOnlyBlob) (*ocmdescriptorruntime.Resource, error) {
	return nil, nil
}
func (m *mockRepo) GetLocalResource(_ context.Context, _, _ string,
	_ ocmruntime.Identity) (ocmblob.ReadOnlyBlob, *ocmdescriptorruntime.Resource, error) {
	return nil, nil, nil
}
func (m *mockRepo) AddLocalSource(_ context.Context, _, _ string, _ *ocmdescriptorruntime.Source,
	_ ocmblob.ReadOnlyBlob) (*ocmdescriptorruntime.Source, error) {
	return nil, nil
}
func (m *mockRepo) GetLocalSource(_ context.Context, _, _ string,
	_ ocmruntime.Identity) (ocmblob.ReadOnlyBlob, *ocmdescriptorruntime.Source, error) {
	return nil, nil, nil
}

// mockSigningHandler implements ocmsigning.Handler.
type mockSigningHandler struct {
	signErr                error
	getConsumerIdentityErr error
}

func (m *mockSigningHandler) GetSigningCredentialConsumerIdentity(_ context.Context, _ string, _ ocmdescriptorruntime.Digest,
	_ ocmruntime.Typed) (ocmruntime.Identity, error) {
	if m.getConsumerIdentityErr != nil {
		return nil, m.signErr
	}
	return make(ocmruntime.Identity), nil
}
func (m *mockSigningHandler) Sign(_ context.Context, _ ocmdescriptorruntime.Digest, _ ocmruntime.Typed,
	_ ocmruntime.Typed) (ocmdescriptorruntime.SignatureInfo, error) {
	if m.signErr != nil {
		return ocmdescriptorruntime.SignatureInfo{}, m.signErr
	}
	return ocmdescriptorruntime.SignatureInfo{Value: "mocksig"}, nil
}
func (m *mockSigningHandler) GetVerifyingCredentialConsumerIdentity(_ context.Context, _ ocmdescriptorruntime.Signature,
	_ ocmruntime.Typed) (ocmruntime.Identity, error) {
	return nil, nil
}
func (m *mockSigningHandler) Verify(_ context.Context, _ ocmdescriptorruntime.Signature, _ ocmruntime.Typed, _ ocmruntime.Typed) error {
	return nil
}

type mockCredentialsResolver struct {
	resolveErr error
}

func (m mockCredentialsResolver) Resolve(_ context.Context, _ ocmruntime.Identity) (ocmruntime.Typed, error) {
	if m.resolveErr != nil {
		return nil, m.resolveErr
	}
	return &mockCredentials{}, nil
}

type mockCredentials struct{}

func (m *mockCredentials) GetType() ocmruntime.Type        { return ocmruntime.Type{} }
func (m *mockCredentials) SetType(_ ocmruntime.Type)       {}
func (m *mockCredentials) DeepCopyTyped() ocmruntime.Typed { return &mockCredentials{} }

var _ = Describe("Sign", func() {
	var (
		ctx            context.Context
		ocmConfig      ocmgenericspecv1.Config
		signingProps   SigningProperties
		testDescriptor *ocmdescriptorruntime.Descriptor
		testDigest     *ocmdescriptorruntime.Digest
	)

	BeforeEach(func() {
		ctx = context.Background()
		ocmConfig = ocmgenericspecv1.Config{}
		testDescriptor = &ocmdescriptorruntime.Descriptor{}
		testDigest = &ocmdescriptorruntime.Digest{
			HashAlgorithm:          "SHA-256",
			NormalisationAlgorithm: "OciArtifactDigest/v1",
			Value:                  "abc123",
		}
		signingProps = SigningProperties{
			ComponentVersion:       "ctf::testdata/ctf//github.com/test/component:1.0.0",
			SignatureName:          "test-sig",
			NormalizationAlgorithm: "OciArtifactDigest/v1",
			HashAlgorithm:          "SHA-256",
			DryRun:                 false,
		}

		ocmGetPluginManager = func(_ context.Context, _ *ocmgenericspecv1.Config) (*manager.PluginManager, error) {
			return &manager.PluginManager{}, nil
		}
		ocmGetCredentialGraph = func(_ context.Context, _ *manager.PluginManager, _ *ocmgenericspecv1.Config) (credentials.Resolver, error) {
			return &mockCredentialsResolver{}, nil
		}
		ocmParseComponentReference = func(_ string, _ ...ocmcompref.Option) (*ocmcompref.Ref, error) {
			ref, _ := ocmcompref.Parse("ctf::testdata/ctf//github.com/test/component:1.0.0",
				ocmcompref.WithCTFAccessMode(ocmrepositoryctfv1.AccessModeReadWrite))
			return ref, nil
		}
		ocmNewComponentRepositoryResolver = func(_ context.Context, _ repository.ComponentVersionRepositoryProvider,
			_ credentials.Resolver, _ ...RepositoryResolverOption) (resolvers.ComponentVersionRepositoryResolver, error) {
			return &mockRepoResolver{repo: &mockRepo{cv: testDescriptor}}, nil
		}
		ocmGenerateDigestForSigning = func(_ context.Context, _ *ocmdescriptorruntime.Descriptor,
			_ *slog.Logger, _, _ string) (*ocmdescriptorruntime.Digest, error) {
			return testDigest, nil
		}
		ocmGetSigningHandler = func(_ context.Context, _ *ocmsigninghandler.SigningRegistry, _ ocmruntime.Typed) (ocmsigning.Handler, error) {
			return &mockSigningHandler{}, nil
		}
	})

	Context("happy path", func() {
		It("persists the signature when DryRun is false", func() {
			repo := &mockRepo{cv: testDescriptor}
			ocmNewComponentRepositoryResolver = func(_ context.Context, _ repository.ComponentVersionRepositoryProvider,
				_ credentials.Resolver, _ ...RepositoryResolverOption) (resolvers.ComponentVersionRepositoryResolver, error) {
				return &mockRepoResolver{repo: repo}, nil
			}
			sig, err := Sign(ctx, signingProps, &ocmConfig)
			Expect(err).ToNot(HaveOccurred())
			Expect(sig).ToNot(BeNil())
			Expect(sig.Name).To(Equal("test-sig"))
			Expect(sig.Digest).To(Equal(*testDigest))
			Expect(repo.addCalled).To(BeTrue())
		})

		It("does not persist the signature when DryRun is true", func() {
			repo := &mockRepo{cv: testDescriptor}
			ocmNewComponentRepositoryResolver = func(_ context.Context, _ repository.ComponentVersionRepositoryProvider,
				_ credentials.Resolver, _ ...RepositoryResolverOption) (resolvers.ComponentVersionRepositoryResolver, error) {
				return &mockRepoResolver{repo: repo}, nil
			}
			signingProps.DryRun = true
			sig, err := Sign(ctx, signingProps, &ocmConfig)
			Expect(err).ToNot(HaveOccurred())
			Expect(sig).ToNot(BeNil())
			Expect(sig.Name).To(Equal("test-sig"))
			Expect(sig.Digest).To(Equal(*testDigest))
			Expect(repo.addCalled).To(BeFalse())
		})

		It("overwrites signature if same exists and OverWriteSignature is true", func() {
			testDescriptor.Signatures = []ocmdescriptorruntime.Signature{
				{Name: "test-sig"},
			}

			signingProps.OverwriteSignatures = true
			signingProps.SignatureName = "test-sig"

			ocmGenerateDigestForSigning = func(_ context.Context, _ *ocmdescriptorruntime.Descriptor,
				_ *slog.Logger, _, _ string) (*ocmdescriptorruntime.Digest, error) {
				return &ocmdescriptorruntime.Digest{HashAlgorithm: "SHA-512"}, nil
			}

			signature, err := Sign(ctx, signingProps, &ocmConfig)
			Expect(err).ToNot(HaveOccurred())
			Expect(signature.Digest).ToNot(Equal(*testDigest))
		})

		It("appends new signature if it does not exist yet", func() {
			testDescriptor.Signatures = []ocmdescriptorruntime.Signature{
				{Name: "test-sig"},
			}

			signingProps.OverwriteSignatures = true
			signingProps.SignatureName = "new-sig"

			_, err := Sign(ctx, signingProps, &ocmConfig)
			Expect(err).ToNot(HaveOccurred())
			Expect(testDescriptor.Signatures).To(HaveLen(2))
			Expect(testDescriptor.Signatures[0].Name).To(Equal("test-sig"))
			Expect(testDescriptor.Signatures[1].Name).To(Equal("new-sig"))
		})
	})

	Context("with failing OCM library methods", func() {
		It("returns an error when getting the plugin manager fails", func() {
			ocmGetPluginManager = func(_ context.Context, _ *ocmgenericspecv1.Config) (*manager.PluginManager, error) {
				return nil, fmt.Errorf("plugin manager unavailable")
			}
			_, err := Sign(ctx, signingProps, &ocmConfig)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get plugin manager"))
		})

		It("returns an error when getting the credential graph fails", func() {
			ocmGetCredentialGraph = func(_ context.Context, _ *manager.PluginManager, _ *ocmgenericspecv1.Config) (credentials.Resolver, error) {
				return nil, fmt.Errorf("credential graph unavailable")
			}
			_, err := Sign(ctx, signingProps, &ocmConfig)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get credential graph"))
		})

		It("returns an error when parsing the component version reference fails", func() {
			ocmParseComponentReference = func(_ string, _ ...ocmcompref.Option) (*ocmcompref.Ref, error) {
				return nil, fmt.Errorf("invalid reference")
			}
			_, err := Sign(ctx, signingProps, &ocmConfig)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to parse component version"))
		})

		It("returns an error when creating the repository resolver fails", func() {
			ocmNewComponentRepositoryResolver = func(_ context.Context, _ repository.ComponentVersionRepositoryProvider,
				_ credentials.Resolver, _ ...RepositoryResolverOption) (resolvers.ComponentVersionRepositoryResolver, error) {
				return nil, fmt.Errorf("resolver unavailable")
			}
			_, err := Sign(ctx, signingProps, &ocmConfig)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to initialize ocm repository"))
		})

		It("returns an error when the resolver cannot find a repository for the component", func() {
			ocmNewComponentRepositoryResolver = func(_ context.Context, _ repository.ComponentVersionRepositoryProvider,
				_ credentials.Resolver, _ ...RepositoryResolverOption) (resolvers.ComponentVersionRepositoryResolver, error) {
				return &mockRepoResolver{err: fmt.Errorf("component not found")}, nil
			}
			_, err := Sign(ctx, signingProps, &ocmConfig)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to access ocm repository"))
		})

		It("returns an error when GetComponentVersion fails", func() {
			ocmNewComponentRepositoryResolver = func(_ context.Context, _ repository.ComponentVersionRepositoryProvider,
				_ credentials.Resolver, _ ...RepositoryResolverOption) (resolvers.ComponentVersionRepositoryResolver, error) {
				return &mockRepoResolver{repo: &mockRepo{cvErr: fmt.Errorf("cv not found")}}, nil
			}
			_, err := Sign(ctx, signingProps, &ocmConfig)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get component version"))
		})

		It("return an error when loading the signer spec fails (missing file)", func() {
			signingProps.SignerSpecPath = "invalid"
			_, err := Sign(ctx, signingProps, &ocmConfig)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to load signer specification"))
		})

		It("returns an error when getting the signing handler fails", func() {
			ocmGetSigningHandler = func(_ context.Context, _ *ocmsigninghandler.SigningRegistry, _ ocmruntime.Typed) (ocmsigning.Handler, error) {
				return nil, fmt.Errorf("no handler registered")
			}
			_, err := Sign(ctx, signingProps, &ocmConfig)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to get signature handler"))
		})

		It("returns an error when a signature already exists and overwrite is disabled", func() {
			testDescriptor.Signatures = []ocmdescriptorruntime.Signature{
				{Name: "test-sig"},
			}
			signingProps.OverwriteSignatures = false
			_, err := Sign(ctx, signingProps, &ocmConfig)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`signature "test-sig" already exists`))
		})

		It("returns an error when generating the digest fails", func() {
			ocmGenerateDigestForSigning = func(_ context.Context, _ *ocmdescriptorruntime.Descriptor,
				_ *slog.Logger, _, _ string) (*ocmdescriptorruntime.Digest, error) {
				return nil, fmt.Errorf("digest generation failed")
			}
			_, err := Sign(ctx, signingProps, &ocmConfig)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to generate digest"))
		})

		It("returns an error when resolving the credentials fails", func() {
			ocmGetCredentialGraph = func(_ context.Context, _ *manager.PluginManager, _ *ocmgenericspecv1.Config) (credentials.Resolver, error) {
				return &mockCredentialsResolver{
					resolveErr: fmt.Errorf("failed to resolve credentials"),
				}, nil
			}
			_, err := Sign(ctx, signingProps, &ocmConfig)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to resolve signing credentials"))
		})

		It("returns an error when the signing handler fails to sign", func() {
			ocmGetSigningHandler = func(_ context.Context, _ *ocmsigninghandler.SigningRegistry, _ ocmruntime.Typed) (ocmsigning.Handler, error) {
				return &mockSigningHandler{signErr: fmt.Errorf("signing failed")}, nil
			}
			_, err := Sign(ctx, signingProps, &ocmConfig)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to sign component descriptor"))
		})

		It("returns an error when the updating the component version fails", func() {
			ocmNewComponentRepositoryResolver = func(_ context.Context, _ repository.ComponentVersionRepositoryProvider,
				_ credentials.Resolver, _ ...RepositoryResolverOption) (resolvers.ComponentVersionRepositoryResolver, error) {
				return &mockRepoResolver{repo: &mockRepo{addErr: fmt.Errorf("error"), cv: testDescriptor}}, nil
			}
			_, err := Sign(ctx, signingProps, &ocmConfig)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to update component version"))
		})
	})
})

var _ = Describe("loadSignerSpec", func() {

	Context("happy path", func() {
		It("loads the default signer spec when the path is empty", func() {
			spec, err := loadSignerSpec("")
			Expect(err).ToNot(HaveOccurred())

			specConfig, ok := spec.(*ocmsigningv1alpha1.Config)
			Expect(ok).To(BeTrue())
			Expect(specConfig).ToNot(BeNil())
			Expect(specConfig.SignatureAlgorithm).To(Equal(ocmsigningv1alpha1.AlgorithmRSASSAPSS))
			Expect(specConfig.SignatureEncodingPolicy).To(Equal(ocmsigningv1alpha1.SignatureEncodingPolicyPlain))
		})
		It("loads the signer spec successfully", func() {
			tempDir := GinkgoT().TempDir()
			configFilePath := filepath.Join(tempDir, "kden", "signer.yaml")
			err := os.MkdirAll(filepath.Dir(configFilePath), os.ModePerm)
			Expect(err).ToNot(HaveOccurred())

			var signerSpecContent = `
type: Any
signatureAlgorithm: Any
signatureEncodingPolicy: Any
`
			err = os.WriteFile(configFilePath, []byte(signerSpecContent), 0644)
			Expect(err).ToNot(HaveOccurred())
			spec, err := loadSignerSpec(configFilePath)
			Expect(err).ToNot(HaveOccurred())

			specConfig, ok := spec.(*ocmruntime.Raw)
			Expect(ok).To(BeTrue())
			Expect(specConfig).ToNot(BeNil())
			data := specConfig.Data
			specAsMap := make(map[string]interface{})
			err = yaml.Unmarshal(data, &specAsMap)
			Expect(err).ToNot(HaveOccurred())
			Expect(specAsMap["type"]).To(BeEquivalentTo("Any"))
			Expect(specAsMap["signatureAlgorithm"]).To(BeEquivalentTo("Any"))
			Expect(specAsMap["signatureEncodingPolicy"]).To(BeEquivalentTo("Any"))
		})
	})

	Context("error cases", func() {
		It("returns an error when the file does not exist", func() {
			_, err := loadSignerSpec("invalid")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring(`failed to read signer spec`))
		})

		It("returns an error when the decoding of the signer spec fails", func() {
			tempDir := GinkgoT().TempDir()
			configFilePath := filepath.Join(tempDir, "kden", "signer.yaml")
			err := os.MkdirAll(filepath.Dir(configFilePath), os.ModePerm)
			Expect(err).ToNot(HaveOccurred())

			var signerSpecContent = `` // empty file causes an error
			err = os.WriteFile(configFilePath, []byte(signerSpecContent), 0644)
			Expect(err).ToNot(HaveOccurred())
			_, err = loadSignerSpec(configFilePath)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("failed to decode signer spec"))
		})
	})
})

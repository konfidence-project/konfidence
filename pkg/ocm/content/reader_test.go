package content_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	kompref "github.com/konfidence-project/konfidence/pkg/ocm/compref"
	"github.com/konfidence-project/konfidence/pkg/ocm/content"
	"github.com/konfidence-project/konfidence/pkg/ocm/content/mocks"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	pkgocm "github.com/konfidence-project/konfidence/pkg/ocm/repository"
	"github.com/konfidence-project/konfidence/pkg/ocm/util"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"ocm.software/open-component-model/bindings/go/blob/inmemory"
	"ocm.software/open-component-model/bindings/go/credentials"
	descruntime "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	"ocm.software/open-component-model/bindings/go/oci/compref"
)

// newTestReader constructs a Reader with a mock client, a no-op verifier, and no
// signature specs (verification disabled).
func newTestReader(mockClient *mocks.MockClient) content.Reader {
	return content.NewReader(mockClient, &noopVerifier{}, nil, nil, nil)
}

// noopVerifier satisfies crypto.Verifier and always succeeds.
type noopVerifier struct{}

func (noopVerifier) Verify(_ context.Context, _ credentials.Resolver, _ []crypto.SignatureSpec, _ []*descruntime.Descriptor) error {
	return nil
}

// testRef builds an upstream compref.Ref for use in mock expectations.
// It goes through kompref.Parse so the Repository.Type field is populated
// identically to what the reader does at runtime.
func testRef(component, version string) compref.Ref {
	raw := fmt.Sprintf("registry.example.com//%s:%s", component, version)
	parsed, err := kompref.Parse(raw)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	return *parsed
}

func mustMarshal(v any) []byte {
	data, err := json.Marshal(v)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	return data
}

var _ = Describe("Reader", func() {
	var (
		mockCtrl   *gomock.Controller
		mockClient *mocks.MockClient
		reader     content.Reader
		ctx        context.Context
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient = mocks.NewMockClient(mockCtrl)
		reader = newTestReader(mockClient)
		ctx = context.Background()
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("GetVectorContent", func() {
		const ref = "registry.example.com//github.com/example/my-vector:1.0.0"

		Context("when the OCM client returns a valid descriptor with references", func() {
			It("returns VectorContent with all component references", func() {
				descriptor := descruntime.Descriptor{
					Component: descruntime.Component{
						ComponentMeta: descruntime.ComponentMeta{
							ObjectMeta: descruntime.ObjectMeta{Name: "github.com/example/my-vector", Version: "1.0.0"},
						},
						References: []descruntime.Reference{
							{
								ElementMeta: descruntime.ElementMeta{
									ObjectMeta: descruntime.ObjectMeta{Name: "svc-a", Version: "0.1.0"},
								},
								Component: "github.com/example/service-a",
							},
							{
								ElementMeta: descruntime.ElementMeta{
									ObjectMeta: descruntime.ObjectMeta{Name: "svc-b", Version: "0.2.0"},
								},
								Component: "github.com/example/service-b",
							},
						},
					},
				}
				mockClient.EXPECT().Get(gomock.Any(), testRef("github.com/example/my-vector", "1.0.0")).Return(descriptor, nil)

				vc, err := reader.GetVectorContent(ctx, ref)

				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(vc.Components).To(gomega.HaveLen(2))
				gomega.Expect(vc.Components[0].ComponentName).To(gomega.Equal("github.com/example/service-a"))
				gomega.Expect(vc.Components[0].Version).To(gomega.Equal("0.1.0"))
				gomega.Expect(vc.Components[1].ComponentName).To(gomega.Equal("github.com/example/service-b"))
				gomega.Expect(vc.Components[1].Version).To(gomega.Equal("0.2.0"))
				gomega.Expect(vc.Config).To(gomega.BeNil())
			})
		})

		Context("when the descriptor carries a vector-config resource", func() {
			It("populates Config with the parsed features and authored blobs", func() {
				configBlob := mustMarshal(map[string]any{
					"features": map[string]any{"dark-mode": map[string]any{"enabled": true}},
					"authored": map[string]any{"replicas": float64(3)},
				})

				configResource := descruntime.Resource{
					ElementMeta: descruntime.ElementMeta{
						ObjectMeta: descruntime.ObjectMeta{Name: util.KonfidenceResourceNameVectorConfig, Version: "1.0.0"},
					},
					Type: "plain-text",
				}
				descriptor := descruntime.Descriptor{
					Component: descruntime.Component{
						ComponentMeta: descruntime.ComponentMeta{
							ObjectMeta: descruntime.ObjectMeta{Name: "github.com/example/my-vector", Version: "1.0.0"},
						},
						Resources: []descruntime.Resource{configResource},
					},
				}
				mockClient.EXPECT().Get(gomock.Any(), testRef("github.com/example/my-vector", "1.0.0")).Return(descriptor, nil)
				mockClient.EXPECT().GetLocalResource(gomock.Any(), testRef("github.com/example/my-vector", "1.0.0"), gomock.Any()).
					Return(inmemory.New(bytes.NewReader(configBlob)), &configResource, nil)

				vc, err := reader.GetVectorContent(ctx, ref)

				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(vc.Config).ToNot(gomega.BeNil())
				gomega.Expect(vc.Config.Features).To(gomega.HaveKey("dark-mode"))
				gomega.Expect(vc.Config.Authored).To(gomega.HaveKeyWithValue("replicas", float64(3)))
			})
		})

		Context("when the descriptor has more than one vector-config resource", func() {
			It("returns an error", func() {
				r1 := descruntime.Resource{
					ElementMeta: descruntime.ElementMeta{
						ObjectMeta: descruntime.ObjectMeta{Name: util.KonfidenceResourceNameVectorConfig, Version: "1.0.0"},
					},
				}
				r2 := descruntime.Resource{
					ElementMeta: descruntime.ElementMeta{
						ObjectMeta: descruntime.ObjectMeta{Name: util.KonfidenceResourceNameVectorConfig, Version: "2.0.0"},
					},
				}
				descriptor := descruntime.Descriptor{
					Component: descruntime.Component{
						ComponentMeta: descruntime.ComponentMeta{
							ObjectMeta: descruntime.ObjectMeta{Name: "github.com/example/my-vector", Version: "1.0.0"},
						},
						Resources: []descruntime.Resource{r1, r2},
					},
				}
				mockClient.EXPECT().Get(gomock.Any(), gomock.Any()).Return(descriptor, nil)

				_, err := reader.GetVectorContent(ctx, ref)

				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(err.Error()).To(gomega.ContainSubstring("more than one resource"))
			})
		})

		Context("when the OCM client returns not-found", func() {
			It("propagates the error wrapping ErrNotFound", func() {
				mockClient.EXPECT().Get(gomock.Any(), gomock.Any()).Return(descruntime.Descriptor{}, pkgocm.ErrNotFound)

				_, err := reader.GetVectorContent(ctx, ref)

				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(errors.Is(err, pkgocm.ErrNotFound)).To(gomega.BeTrue())
			})
		})

		Context("when the OCM client returns a generic error", func() {
			It("wraps and returns the error", func() {
				mockClient.EXPECT().Get(gomock.Any(), gomock.Any()).Return(descruntime.Descriptor{}, fmt.Errorf("registry unreachable"))

				_, err := reader.GetVectorContent(ctx, ref)

				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(err.Error()).To(gomega.ContainSubstring("registry unreachable"))
			})
		})
	})

	Describe("GetArtifactContent", func() {
		const ref = "registry.example.com//github.com/example/my-artifact:2.0.0"

		Context("when the descriptor has a valid artifact manifest and tasks", func() {
			It("returns ArtifactContent with type, allowReuse, tasks, and resources", func() {
				manifestResource := descruntime.Resource{
					ElementMeta: descruntime.ElementMeta{
						ObjectMeta: descruntime.ObjectMeta{Name: "manifest", Version: "1.0.0"},
					},
					Type: "cloud.konfidence.artifact.manifest",
				}
				task1Resource := descruntime.Resource{
					ElementMeta: descruntime.ElementMeta{
						ObjectMeta: descruntime.ObjectMeta{Name: "task-deploy", Version: "1.0.0"},
					},
					Type: "cloud.konfidence.artifact.task.manifest",
				}
				helmResource := descruntime.Resource{
					ElementMeta: descruntime.ElementMeta{
						ObjectMeta: descruntime.ObjectMeta{Name: "helm-chart", Version: "1.0.0"},
					},
					Type: "helmChart",
				}

				manifestJSON := mustMarshal(map[string]any{"type": "cloud.konfidence.flux.helm", "allowReuse": true})
				task1JSON := mustMarshal(map[string]any{
					"name":      "deploy",
					"type":      "flux-helm-release",
					"dependsOn": []string{"pre-deploy"},
				})

				descriptor := descruntime.Descriptor{
					Component: descruntime.Component{
						ComponentMeta: descruntime.ComponentMeta{
							ObjectMeta: descruntime.ObjectMeta{Name: "github.com/example/my-artifact", Version: "2.0.0"},
						},
						Resources: []descruntime.Resource{manifestResource, task1Resource, helmResource},
					},
				}
				mockClient.EXPECT().Get(gomock.Any(), testRef("github.com/example/my-artifact", "2.0.0")).Return(descriptor, nil)
				mockClient.EXPECT().GetLocalResource(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(inmemory.New(bytes.NewReader(manifestJSON)), &manifestResource, nil)
				mockClient.EXPECT().GetLocalResource(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(inmemory.New(bytes.NewReader(task1JSON)), &task1Resource, nil)

				ac, err := reader.GetArtifactContent(ctx, ref)

				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(ac.Type).To(gomega.Equal("cloud.konfidence.flux.helm"))
				gomega.Expect(ac.AllowReuse).To(gomega.BeTrue())
				gomega.Expect(ac.Tasks).To(gomega.HaveLen(1))
				gomega.Expect(ac.Tasks[0].Name).To(gomega.Equal("deploy"))
				gomega.Expect(ac.Tasks[0].Type).To(gomega.Equal("flux-helm-release"))
				gomega.Expect(ac.Tasks[0].DependsOn).To(gomega.ConsistOf("pre-deploy"))
				gomega.Expect(ac.Resources).To(gomega.HaveLen(1))
				gomega.Expect(ac.Resources[0].Name).To(gomega.Equal("helm-chart"))
				gomega.Expect(ac.Resources[0].Type).To(gomega.Equal("helmChart"))
			})
		})

		Context("when the descriptor has no artifact manifest resource", func() {
			It("returns an error", func() {
				descriptor := descruntime.Descriptor{
					Component: descruntime.Component{
						ComponentMeta: descruntime.ComponentMeta{
							ObjectMeta: descruntime.ObjectMeta{Name: "github.com/example/my-artifact", Version: "2.0.0"},
						},
						Resources: []descruntime.Resource{},
					},
				}
				mockClient.EXPECT().Get(gomock.Any(), gomock.Any()).Return(descriptor, nil)

				_, err := reader.GetArtifactContent(ctx, ref)

				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(err.Error()).To(gomega.ContainSubstring("no artifact manifest resource found"))
			})
		})

		Context("when the manifest blob is malformed JSON", func() {
			It("returns an unmarshal error", func() {
				manifestResource := descruntime.Resource{
					ElementMeta: descruntime.ElementMeta{
						ObjectMeta: descruntime.ObjectMeta{Name: "manifest", Version: "1.0.0"},
					},
					Type: "cloud.konfidence.artifact.manifest",
				}
				descriptor := descruntime.Descriptor{
					Component: descruntime.Component{
						ComponentMeta: descruntime.ComponentMeta{
							ObjectMeta: descruntime.ObjectMeta{Name: "github.com/example/my-artifact", Version: "2.0.0"},
						},
						Resources: []descruntime.Resource{manifestResource},
					},
				}
				mockClient.EXPECT().Get(gomock.Any(), gomock.Any()).Return(descriptor, nil)
				mockClient.EXPECT().GetLocalResource(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(inmemory.New(bytes.NewReader([]byte("not-json"))), &manifestResource, nil)

				_, err := reader.GetArtifactContent(ctx, ref)

				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(err.Error()).To(gomega.ContainSubstring("unmarshal artifact manifest"))
			})
		})

		Context("when the OCM client returns not-found", func() {
			It("propagates the error wrapping ErrNotFound", func() {
				mockClient.EXPECT().Get(gomock.Any(), gomock.Any()).Return(descruntime.Descriptor{}, pkgocm.ErrNotFound)

				_, err := reader.GetArtifactContent(ctx, ref)

				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(errors.Is(err, pkgocm.ErrNotFound)).To(gomega.BeTrue())
			})
		})

		Context("when the task manifest blob is malformed JSON", func() {
			It("returns an unmarshal error", func() {
				manifestResource := descruntime.Resource{
					ElementMeta: descruntime.ElementMeta{
						ObjectMeta: descruntime.ObjectMeta{Name: "manifest", Version: "1.0.0"},
					},
					Type: "cloud.konfidence.artifact.manifest",
				}
				taskResource := descruntime.Resource{
					ElementMeta: descruntime.ElementMeta{
						ObjectMeta: descruntime.ObjectMeta{Name: "task-bad", Version: "1.0.0"},
					},
					Type: "cloud.konfidence.artifact.task.manifest",
				}
				manifestJSON := mustMarshal(map[string]any{"type": "cloud.konfidence.flux.helm", "allowReuse": false})
				descriptor := descruntime.Descriptor{
					Component: descruntime.Component{
						ComponentMeta: descruntime.ComponentMeta{
							ObjectMeta: descruntime.ObjectMeta{Name: "github.com/example/my-artifact", Version: "2.0.0"},
						},
						Resources: []descruntime.Resource{manifestResource, taskResource},
					},
				}
				mockClient.EXPECT().Get(gomock.Any(), gomock.Any()).Return(descriptor, nil)
				mockClient.EXPECT().GetLocalResource(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(inmemory.New(bytes.NewReader(manifestJSON)), &manifestResource, nil)
				mockClient.EXPECT().GetLocalResource(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(inmemory.New(bytes.NewReader([]byte("not-json"))), &taskResource, nil)

				_, err := reader.GetArtifactContent(ctx, ref)

				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(err.Error()).To(gomega.ContainSubstring("unmarshal task manifest"))
			})
		})
	})
})

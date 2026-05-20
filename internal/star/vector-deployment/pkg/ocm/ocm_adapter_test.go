package ocm_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"ocm.software/open-component-model/bindings/go/blob/inmemory"
	descruntime "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	ociv1 "ocm.software/open-component-model/bindings/go/oci/spec/repository/v1/oci"
	ocmRuntime "ocm.software/open-component-model/bindings/go/runtime"

	ocmadapter "github.com/konfidence-project/konfidence/internal/star/vector-deployment/pkg/ocm"
	"github.com/konfidence-project/konfidence/internal/star/vector-deployment/pkg/ocm/mocks"
)

// newTestAdapter creates an Adapter with the given mock client injected via the exported constructor
// for testing. We bypass the OCI builder by constructing through the internal field directly.
func newTestAdapter(ctrl *gomock.Controller) (*mocks.MockClient, ocmadapter.Adapter) {
	mock := mocks.NewMockClient(ctrl)
	return mock, ocmadapter.NewAdapterWithClient(mock)
}

var _ = Describe("OcmAdapter", func() {
	var (
		mockCtrl   *gomock.Controller
		mockClient *mocks.MockClient
		adapter    ocmadapter.Adapter
		ctx        context.Context
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockClient, adapter = newTestAdapter(mockCtrl)
		ctx = context.Background()
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Describe("GetVectorDescriptor", func() {
		var vectorRef compref.Ref

		BeforeEach(func() {
			vectorRef = compref.Ref{
				Repository: &ociv1.Repository{BaseUrl: "https://registry.example.com"},
				Component:  "github.com/example/my-vector",
				Version:    "1.0.0",
			}
		})

		Context("when the OCM client returns a valid descriptor", func() {
			It("returns a VectorDescriptor with DescriptorJSON and References populated", func() {
				descriptor := descruntime.Descriptor{
					Component: descruntime.Component{
						ComponentMeta: descruntime.ComponentMeta{
							ObjectMeta: descruntime.ObjectMeta{
								Name:    "github.com/example/my-vector",
								Version: "1.0.0",
							},
						},
						Provider: descruntime.Provider{Name: "example-org"},
						References: []descruntime.Reference{
							{
								ElementMeta: descruntime.ElementMeta{
									ObjectMeta: descruntime.ObjectMeta{
										Name:    "service-a",
										Version: "0.1.0",
									},
								},
								Component: "github.com/example/service-a",
							},
							{
								ElementMeta: descruntime.ElementMeta{
									ObjectMeta: descruntime.ObjectMeta{
										Name:    "service-b",
										Version: "0.2.0",
									},
								},
								Component: "github.com/example/service-b",
							},
						},
					},
				}
				mockClient.EXPECT().Get(gomock.Any(), vectorRef).Return(descriptor, nil)

				vd, err := adapter.GetVectorDescriptor(ctx, vectorRef)

				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				By("setting DescriptorJSON to a non-empty JSON string")
				gomega.Expect(vd.DescriptorJSON).ToNot(gomega.BeEmpty())
				gomega.Expect(string(vd.DescriptorJSON)).To(gomega.ContainSubstring("github.com/example/my-vector"))

				By("mapping component references to References with the vector's repository")
				gomega.Expect(vd.References).To(gomega.HaveLen(2))
				gomega.Expect(vd.References[0].Component).To(gomega.Equal("github.com/example/service-a"))
				gomega.Expect(vd.References[0].Version).To(gomega.Equal("0.1.0"))
				gomega.Expect(vd.References[0].Repository).To(gomega.Equal(vectorRef.Repository))
				gomega.Expect(vd.References[1].Component).To(gomega.Equal("github.com/example/service-b"))
				gomega.Expect(vd.References[1].Version).To(gomega.Equal("0.2.0"))
			})

			It("returns a VectorDescriptor with empty References when the descriptor has no references", func() {
				descriptor := descruntime.Descriptor{
					Component: descruntime.Component{
						ComponentMeta: descruntime.ComponentMeta{
							ObjectMeta: descruntime.ObjectMeta{
								Name:    "github.com/example/my-vector",
								Version: "1.0.0",
							},
						},
						Provider: descruntime.Provider{Name: "example-org"},
					},
				}
				mockClient.EXPECT().Get(gomock.Any(), vectorRef).Return(descriptor, nil)

				vd, err := adapter.GetVectorDescriptor(ctx, vectorRef)

				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(vd.References).To(gomega.BeEmpty())
			})
		})

		Context("when the OCM client returns an error", func() {
			It("propagates the error", func() {
				mockClient.EXPECT().Get(gomock.Any(), vectorRef).Return(descruntime.Descriptor{}, errors.New("registry unavailable"))

				_, err := adapter.GetVectorDescriptor(ctx, vectorRef)

				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(err.Error()).To(gomega.ContainSubstring("registry unavailable"))
			})
		})
	})

	Describe("GetArtifactManifestByReference", func() {
		var artifactRef compref.Ref

		BeforeEach(func() {
			artifactRef = compref.Ref{
				Repository: &ociv1.Repository{BaseUrl: "https://registry.example.com"},
				Component:  "github.com/example/service-a",
				Version:    "0.1.0",
			}
		})

		// helper: build a descriptor with the given resources
		buildDescriptor := func(resources []descruntime.Resource) descruntime.Descriptor {
			return descruntime.Descriptor{
				Component: descruntime.Component{
					ComponentMeta: descruntime.ComponentMeta{
						ObjectMeta: descruntime.ObjectMeta{
							Name:    artifactRef.Component,
							Version: artifactRef.Version,
						},
					},
					Provider:  descruntime.Provider{Name: "example-org"},
					Resources: resources,
				},
			}
		}

		Context("with a valid manifest resource and task manifest resources", func() {
			It("returns an ArtifactManifest with tasks and non-blob resources populated", func() {
				manifestJSON := []byte(`{"type":"cloud.konfidence.flux.helm","allowReuse":true}`)
				task1JSON := []byte(`{"name":"migrate-db","type":"k8s-job","dependsOn":[],"spec":{"image":"migrate:1.0"}}`)
				task2JSON := []byte(`{"name":"seed-data","type":"k8s-job","dependsOn":["migrate-db"],"spec":{"image":"seed:1.0"}}`)

				manifestResource := descruntime.Resource{
					ElementMeta: descruntime.ElementMeta{ObjectMeta: descruntime.ObjectMeta{Name: "artifact-manifest"}},
					Type:        "cloud.konfidence.artifact.manifest",
				}
				task1Resource := descruntime.Resource{
					ElementMeta: descruntime.ElementMeta{ObjectMeta: descruntime.ObjectMeta{Name: "task-migrate-db"}},
					Type:        "cloud.konfidence.artifact.task.manifest",
				}
				task2Resource := descruntime.Resource{
					ElementMeta: descruntime.ElementMeta{ObjectMeta: descruntime.ObjectMeta{Name: "task-seed-data"}},
					Type:        "cloud.konfidence.artifact.task.manifest",
				}

				descriptor := buildDescriptor([]descruntime.Resource{manifestResource, task1Resource, task2Resource})

				mockClient.EXPECT().Get(gomock.Any(), artifactRef).Return(descriptor, nil)
				mockClient.EXPECT().GetLocalResource(gomock.Any(), artifactRef, manifestResource.ToIdentity()).
					Return(inmemory.New(bytes.NewReader(manifestJSON)), &manifestResource, nil)
				mockClient.EXPECT().GetLocalResource(gomock.Any(), artifactRef, task1Resource.ToIdentity()).
					Return(inmemory.New(bytes.NewReader(task1JSON)), &task1Resource, nil)
				mockClient.EXPECT().GetLocalResource(gomock.Any(), artifactRef, task2Resource.ToIdentity()).
					Return(inmemory.New(bytes.NewReader(task2JSON)), &task2Resource, nil)

				manifest, err := adapter.GetArtifactManifestByReference(ctx, artifactRef)

				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				By("populating manifest fields from the blob")
				gomega.Expect(manifest.Type).To(gomega.Equal("cloud.konfidence.flux.helm"))
				gomega.Expect(manifest.AllowReuse).To(gomega.BeTrue())

				By("populating tasks from task manifest blobs")
				gomega.Expect(manifest.Tasks).To(gomega.HaveLen(2))
				gomega.Expect(manifest.Tasks[0].Name).To(gomega.Equal("migrate-db"))
				gomega.Expect(manifest.Tasks[0].Type).To(gomega.Equal("k8s-job"))
				gomega.Expect(manifest.Tasks[0].DependsOn).To(gomega.BeEmpty())
				gomega.Expect(manifest.Tasks[1].Name).To(gomega.Equal("seed-data"))
				gomega.Expect(manifest.Tasks[1].DependsOn).To(gomega.ConsistOf("migrate-db"))
			})
		})

		Context("when a descriptor has no artifact manifest resource", func() {
			It("returns an error", func() {
				descriptor := buildDescriptor([]descruntime.Resource{})
				mockClient.EXPECT().Get(gomock.Any(), artifactRef).Return(descriptor, nil)

				_, err := adapter.GetArtifactManifestByReference(ctx, artifactRef)

				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(err.Error()).To(gomega.ContainSubstring("no artifact manifest found"))
			})
		})

		Context("when the OCM client Get returns an error", func() {
			It("propagates the error", func() {
				mockClient.EXPECT().Get(gomock.Any(), artifactRef).Return(descruntime.Descriptor{}, fmt.Errorf("network timeout"))

				_, err := adapter.GetArtifactManifestByReference(ctx, artifactRef)

				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(err.Error()).To(gomega.ContainSubstring("network timeout"))
			})
		})

		Context("when GetLocalResource returns an error for the manifest blob", func() {
			It("propagates the error", func() {
				manifestResource := descruntime.Resource{
					ElementMeta: descruntime.ElementMeta{ObjectMeta: descruntime.ObjectMeta{Name: "artifact-manifest"}},
					Type:        "cloud.konfidence.artifact.manifest",
				}
				descriptor := buildDescriptor([]descruntime.Resource{manifestResource})

				mockClient.EXPECT().Get(gomock.Any(), artifactRef).Return(descriptor, nil)
				mockClient.EXPECT().GetLocalResource(gomock.Any(), artifactRef, manifestResource.ToIdentity()).
					Return(nil, nil, fmt.Errorf("blob not found"))

				_, err := adapter.GetArtifactManifestByReference(ctx, artifactRef)

				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(err.Error()).To(gomega.ContainSubstring("blob not found"))
			})
		})

		Context("when the manifest blob contains invalid JSON", func() {
			It("returns an unmarshal error", func() {
				manifestResource := descruntime.Resource{
					ElementMeta: descruntime.ElementMeta{ObjectMeta: descruntime.ObjectMeta{Name: "artifact-manifest"}},
					Type:        "cloud.konfidence.artifact.manifest",
				}
				descriptor := buildDescriptor([]descruntime.Resource{manifestResource})

				mockClient.EXPECT().Get(gomock.Any(), artifactRef).Return(descriptor, nil)
				mockClient.EXPECT().GetLocalResource(gomock.Any(), artifactRef, manifestResource.ToIdentity()).
					Return(inmemory.New(bytes.NewReader([]byte(`not-json`))), &manifestResource, nil)

				_, err := adapter.GetArtifactManifestByReference(ctx, artifactRef)

				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(err.Error()).To(gomega.ContainSubstring("failed to unmarshal manifest data"))
			})
		})

		Context("when there is only a manifest resource with no tasks", func() {
			It("returns a manifest with an empty Tasks slice", func() {
				manifestJSON := []byte(`{"type":"cloud.konfidence.oci.image","allowReuse":false}`)
				manifestResource := descruntime.Resource{
					ElementMeta: descruntime.ElementMeta{ObjectMeta: descruntime.ObjectMeta{Name: "artifact-manifest"}},
					Type:        "cloud.konfidence.artifact.manifest",
				}
				descriptor := buildDescriptor([]descruntime.Resource{manifestResource})

				mockClient.EXPECT().Get(gomock.Any(), artifactRef).Return(descriptor, nil)
				mockClient.EXPECT().GetLocalResource(gomock.Any(), artifactRef, manifestResource.ToIdentity()).
					Return(inmemory.New(bytes.NewReader(manifestJSON)), &manifestResource, nil)

				manifest, err := adapter.GetArtifactManifestByReference(ctx, artifactRef)

				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				gomega.Expect(manifest.Type).To(gomega.Equal("cloud.konfidence.oci.image"))
				gomega.Expect(manifest.AllowReuse).To(gomega.BeFalse())
				gomega.Expect(manifest.Tasks).To(gomega.BeEmpty())
			})
		})

		Context("with non-manifest resources (helmChart, ociImage)", func() {
			It("collects their access specs into the Resources slice of the ArtifactManifest", func() {
				manifestJSON := []byte(`{"type":"cloud.konfidence.flux.helm","allowReuse":true}`)
				helmChartJSON := []byte(`{"type":"helm","helmChart":"podinfo:6.9.1","helmRepository":"https://stefanprodan.github.io/podinfo"}`)
				ociImageJSON := []byte(`{"type":"ociArtifact","imageReference":"stefanprodan/podinfo:6.9.1"}`)

				manifestResource := descruntime.Resource{
					ElementMeta: descruntime.ElementMeta{ObjectMeta: descruntime.ObjectMeta{Name: "artifact-manifest"}},
					Type:        "cloud.konfidence.artifact.manifest",
				}
				helmChartResource := descruntime.Resource{
					ElementMeta: descruntime.ElementMeta{ObjectMeta: descruntime.ObjectMeta{Name: "service-helm-chart"}},
					Type:        "helmChart",
					Access:      &ocmRuntime.Raw{Data: helmChartJSON},
				}
				ociImageResource := descruntime.Resource{
					ElementMeta: descruntime.ElementMeta{ObjectMeta: descruntime.ObjectMeta{Name: "service-image"}},
					Type:        "ociImage",
					Access:      &ocmRuntime.Raw{Data: ociImageJSON},
				}

				descriptor := buildDescriptor([]descruntime.Resource{manifestResource, helmChartResource, ociImageResource})

				mockClient.EXPECT().Get(gomock.Any(), artifactRef).Return(descriptor, nil)
				mockClient.EXPECT().GetLocalResource(gomock.Any(), artifactRef, manifestResource.ToIdentity()).
					Return(inmemory.New(bytes.NewReader(manifestJSON)), &manifestResource, nil)

				manifest, err := adapter.GetArtifactManifestByReference(ctx, artifactRef)

				gomega.Expect(err).ToNot(gomega.HaveOccurred())

				By("populating manifest fields correctly")
				gomega.Expect(manifest.Type).To(gomega.Equal("cloud.konfidence.flux.helm"))
				gomega.Expect(manifest.AllowReuse).To(gomega.BeTrue())

				By("collecting non-manifest resources into Resources")
				gomega.Expect(manifest.Resources).To(gomega.HaveLen(2))
				gomega.Expect(manifest.Resources[0].Name).To(gomega.Equal("service-helm-chart"))
				gomega.Expect(manifest.Resources[0].Type).To(gomega.Equal("helmChart"))
				gomega.Expect(manifest.Resources[0].Content).To(gomega.Equal(helmChartJSON))
				gomega.Expect(manifest.Resources[1].Name).To(gomega.Equal("service-image"))
				gomega.Expect(manifest.Resources[1].Type).To(gomega.Equal("ociImage"))
				gomega.Expect(manifest.Resources[1].Content).To(gomega.Equal(ociImageJSON))
			})
		})

		Context("when a non-manifest resource has no access spec", func() {
			It("returns an error", func() {
				manifestJSON := []byte(`{"type":"cloud.konfidence.flux.helm","allowReuse":false}`)

				manifestResource := descruntime.Resource{
					ElementMeta: descruntime.ElementMeta{ObjectMeta: descruntime.ObjectMeta{Name: "artifact-manifest"}},
					Type:        "cloud.konfidence.artifact.manifest",
				}
				helmChartResource := descruntime.Resource{
					ElementMeta: descruntime.ElementMeta{ObjectMeta: descruntime.ObjectMeta{Name: "service-helm-chart"}},
					Type:        "helmChart",
					// Access intentionally nil
				}

				descriptor := buildDescriptor([]descruntime.Resource{manifestResource, helmChartResource})

				mockClient.EXPECT().Get(gomock.Any(), artifactRef).Return(descriptor, nil)
				mockClient.EXPECT().GetLocalResource(gomock.Any(), artifactRef, manifestResource.ToIdentity()).
					Return(inmemory.New(bytes.NewReader(manifestJSON)), &manifestResource, nil)

				_, err := adapter.GetArtifactManifestByReference(ctx, artifactRef)

				gomega.Expect(err).To(gomega.HaveOccurred())
				gomega.Expect(err.Error()).To(gomega.ContainSubstring("missing access spec"))
			})
		})
	})

	Describe("RepositoryURL helper via GetVectorDescriptor", func() {
		It("returns empty string when repository spec is not an OCI repository", func() {
			vectorRef := compref.Ref{
				Repository: ocmRuntime.Identity{"type": "unknown"},
				Component:  "github.com/example/vec",
				Version:    "2.0.0",
			}
			descriptor := descruntime.Descriptor{
				Component: descruntime.Component{
					ComponentMeta: descruntime.ComponentMeta{
						ObjectMeta: descruntime.ObjectMeta{Name: "github.com/example/vec", Version: "2.0.0"},
					},
					Provider: descruntime.Provider{Name: "example-org"},
				},
			}
			mockClient.EXPECT().Get(gomock.Any(), vectorRef).Return(descriptor, nil)

			vd, err := adapter.GetVectorDescriptor(ctx, vectorRef)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(vd.DescriptorJSON).ToNot(gomega.BeEmpty())
		})
	})
})

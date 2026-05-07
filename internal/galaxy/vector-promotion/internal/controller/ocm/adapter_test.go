/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ocm

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"

	"github.com/konfidence-project/gcp-vector-promotion-controller/internal/controller/domain"
	"github.com/konfidence-project/gcp-vector-promotion-controller/internal/controller/ocm/internal/mock"
	descruntime "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	ocispec "ocm.software/open-component-model/bindings/go/oci/spec/repository/v1/oci"
	"ocm.software/open-component-model/bindings/go/runtime"
)

var _ = Describe("PromotionAdapter", func() {
	var (
		ctrl       *gomock.Controller
		mockClient *mock.MockClient
		adapter    *PromotionAdapter
		ctx        context.Context
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		mockClient = mock.NewMockClient(ctrl)
		adapter = NewPromotionAdapter(mockClient)
		ctx = context.Background()
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("Promote", func() {
		Context("with invalid source reference", func() {
			It("returns ErrInvalidConfiguration for malformed source", func() {
				err := adapter.Promote(ctx, "not-a-valid-reference", "ghcr.io/org/components//github.com/org/app:production")

				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, domain.ErrInvalidConfiguration)).To(BeTrue())
				Expect(err.Error()).To(ContainSubstring("failed to parse source reference"))
			})

			It("returns ErrInvalidConfiguration for source without version", func() {
				err := adapter.Promote(ctx, "ghcr.io/org/components//github.com/org/app", "ghcr.io/org/components//github.com/org/app:production")

				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, domain.ErrInvalidConfiguration)).To(BeTrue())
			})
		})

		Context("with invalid target reference", func() {
			It("returns ErrInvalidConfiguration for malformed target", func() {
				err := adapter.Promote(ctx, "ghcr.io/org/components//github.com/org/app:1.0.0", "not-a-valid-reference")

				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, domain.ErrInvalidConfiguration)).To(BeTrue())
				Expect(err.Error()).To(ContainSubstring("failed to parse target reference"))
			})

			It("returns ErrInvalidConfiguration for target with semver instead of alias", func() {
				err := adapter.Promote(ctx, "ghcr.io/org/components//github.com/org/app:1.0.0", "ghcr.io/org/components//github.com/org/app:2.0.0")

				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, domain.ErrInvalidConfiguration)).To(BeTrue())
				Expect(err.Error()).To(ContainSubstring("failed to parse target reference"))
			})
		})

		Context("when source resolution fails", func() {
			It("returns ErrFetchingSourceFailed", func() {
				mockClient.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(descruntime.Descriptor{}, errors.New("source not found"))

				err := adapter.Promote(ctx, "ghcr.io/org/components//github.com/org/app:1.0.0", "ghcr.io/org/components//github.com/org/app:production")

				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, domain.ErrFetchingSourceFailed)).To(BeTrue())
				Expect(err.Error()).To(ContainSubstring("failed to get source reference"))
			})
		})

		Context("when source and target are in the same repository", func() {
			It("skips copy and only adds alias", func() {
				mockClient.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(makeDescriptor("1.0.0"), nil)

				mockClient.EXPECT().
					AddAlias(gomock.Any(), gomock.Any(), "production").
					Return(nil)

				err := adapter.Promote(ctx,
					"ghcr.io/org/components//github.com/org/app:1.0.0",
					"ghcr.io/org/components//github.com/org/app:production")

				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when source and target are in different repositories", func() {
			It("copies and then adds alias", func() {
				mockClient.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(makeDescriptor("1.0.0"), nil)

				mockClient.EXPECT().
					Copy(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)

				mockClient.EXPECT().
					AddAlias(gomock.Any(), gomock.Any(), "production").
					Return(nil)

				err := adapter.Promote(ctx,
					"ghcr.io/org/source-repo//github.com/org/app:1.0.0",
					"ghcr.io/org/target-repo//github.com/org/app:production")

				Expect(err).ToNot(HaveOccurred())
			})

			It("returns error when copy fails", func() {
				mockClient.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(makeDescriptor("1.0.0"), nil)

				mockClient.EXPECT().
					Copy(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errors.New("copy failed"))

				err := adapter.Promote(ctx,
					"ghcr.io/org/source-repo//github.com/org/app:1.0.0",
					"ghcr.io/org/target-repo//github.com/org/app:production")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to copy"))
			})
		})

		Context("when adding alias fails", func() {
			It("returns error", func() {
				mockClient.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(makeDescriptor("1.0.0"), nil)

				mockClient.EXPECT().
					AddAlias(gomock.Any(), gomock.Any(), "production").
					Return(errors.New("alias failed"))

				err := adapter.Promote(ctx,
					"ghcr.io/org/components//github.com/org/app:1.0.0",
					"ghcr.io/org/components//github.com/org/app:production")

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to add alias"))
			})
		})

		Context("when source uses an alias that resolves to a version", func() {
			It("promotes the resolved version", func() {
				mockClient.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, ref compref.Ref) (descruntime.Descriptor, error) {
						Expect(ref.Version).To(Equal("latest"))
						return makeDescriptor("2.5.3"), nil
					})

				mockClient.EXPECT().
					AddAlias(gomock.Any(), gomock.Any(), "production").
					DoAndReturn(func(ctx context.Context, ref compref.Ref, alias string) error {
						Expect(ref.Version).To(Equal("2.5.3"))
						return nil
					})

				err := adapter.Promote(ctx,
					"ghcr.io/org/components//github.com/org/app:latest",
					"ghcr.io/org/components//github.com/org/app:production")

				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})

var _ = Describe("sameLocation", func() {
	DescribeTable("compares OCI repositories",
		func(a, b *compref.Ref, expected bool) {
			result, err := sameLocation(a, b)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(expected))
		},
		Entry("same base url and sub path",
			&compref.Ref{
				Repository: &ocispec.Repository{
					Type:    runtime.Type{Name: ocispec.Type, Version: "v1"},
					BaseUrl: "https://registry.example.com",
					SubPath: "repo",
				},
			},
			&compref.Ref{
				Repository: &ocispec.Repository{
					Type:    runtime.Type{Name: ocispec.Type, Version: "v1"},
					BaseUrl: "https://registry.example.com",
					SubPath: "repo",
				},
			},
			true,
		),
		Entry("different base url",
			&compref.Ref{
				Repository: &ocispec.Repository{
					Type:    runtime.Type{Name: ocispec.Type, Version: "v1"},
					BaseUrl: "https://registry-a.example.com",
					SubPath: "repo",
				},
			},
			&compref.Ref{
				Repository: &ocispec.Repository{
					Type:    runtime.Type{Name: ocispec.Type, Version: "v1"},
					BaseUrl: "https://registry-b.example.com",
					SubPath: "repo",
				},
			},
			false,
		),
		Entry("different sub path",
			&compref.Ref{
				Repository: &ocispec.Repository{
					Type:    runtime.Type{Name: ocispec.Type, Version: "v1"},
					BaseUrl: "https://registry.example.com",
					SubPath: "repo-a",
				},
			},
			&compref.Ref{
				Repository: &ocispec.Repository{
					Type:    runtime.Type{Name: ocispec.Type, Version: "v1"},
					BaseUrl: "https://registry.example.com",
					SubPath: "repo-b",
				},
			},
			false,
		),
	)

	Context("with non-OCI repositories", func() {
		It("returns error when source is not OCI", func() {
			a := &compref.Ref{Repository: &nonOCIRepository{}}
			b := &compref.Ref{Repository: &ocispec.Repository{}}

			_, err := sameLocation(a, b)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("source repository is not an OCI repository"))
		})

		It("returns error when target is not OCI", func() {
			a := &compref.Ref{Repository: &ocispec.Repository{}}
			b := &compref.Ref{Repository: &nonOCIRepository{}}

			_, err := sameLocation(a, b)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("target repository is not an OCI repository"))
		})
	})
})

func makeDescriptor(version string) descruntime.Descriptor {
	return descruntime.Descriptor{
		Component: descruntime.Component{
			ComponentMeta: descruntime.ComponentMeta{
				ObjectMeta: descruntime.ObjectMeta{
					Name:    "github.com/org/app",
					Version: version,
				},
			},
		},
	}
}

type nonOCIRepository struct{}

func (n *nonOCIRepository) GetType() runtime.Type        { return runtime.Type{Name: "other"} }
func (n *nonOCIRepository) SetType(t runtime.Type)       {}
func (n *nonOCIRepository) DeepCopyTyped() runtime.Typed { return &nonOCIRepository{} }

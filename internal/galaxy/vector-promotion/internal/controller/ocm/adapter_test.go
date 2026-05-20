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

	konfcompref "github.com/konfidence-project/konfidence/pkg/ocm/compref"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	descruntime "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	ocispec "ocm.software/open-component-model/bindings/go/oci/spec/repository/v1/oci"
	"ocm.software/open-component-model/bindings/go/runtime"

	"github.com/konfidence-project/konfidence/internal/galaxy/vector-promotion/internal/controller/domain"
	"github.com/konfidence-project/konfidence/internal/galaxy/vector-promotion/internal/controller/ocm/internal/mock"
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
		adapter = NewPromotionAdapter()
		adapter.ocmClient = mockClient
		ctx = context.Background()
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("Promote", func() {
		Context("when source resolution fails", func() {
			It("returns ErrFetchingSourceFailed", func() {
				mockClient.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(descruntime.Descriptor{}, errors.New("source not found"))

				err := adapter.Promote(ctx,
					mustParse("ghcr.io/org/components//github.com/org/app:1.0.0"),
					mustParse("ghcr.io/org/components//github.com/org/app:production"),
				)

				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, domain.ErrFetchingSourceFailed)).To(BeTrue())
				Expect(err.Error()).To(ContainSubstring("failed to get descriptor of source reference"))
			})
		})

		Context("when source vector verification fails", func() {
			It("returns ErrFetchingSourceFailed", func() {
				mockVerifier := mock.NewMockVerifier(ctrl)
				verifyAdapter := NewPromotionAdapter(WithVectorVerifier(mockVerifier))
				verifyAdapter.ocmClient = mockClient

				mockClient.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(makeDescriptor("1.0.0"), nil)

				mockVerifier.EXPECT().
					Verify(gomock.Any(), gomock.Any()).
					Return(errors.New("signature mismatch"))

				err := verifyAdapter.Promote(ctx,
					mustParse("ghcr.io/org/components//github.com/org/app:latest"),
					mustParse("ghcr.io/org/components//github.com/org/app:production"),
				)

				Expect(err).To(HaveOccurred())
				Expect(errors.Is(err, domain.ErrSourceVerificationFailed)).To(BeTrue())
				Expect(err.Error()).To(ContainSubstring("unable to verify signature of the source reference descriptor"))
				Expect(err.Error()).To(ContainSubstring("signature mismatch"))
			})
		})

		Context("when source vector verification succeeds", func() {
			It("proceeds with promotion", func() {
				mockVerifier := mock.NewMockVerifier(ctrl)
				verifyAdapter := NewPromotionAdapter(WithVectorVerifier(mockVerifier))
				verifyAdapter.ocmClient = mockClient

				mockClient.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(makeDescriptor("1.0.0"), nil)

				mockVerifier.EXPECT().
					Verify(gomock.Any(), gomock.Any()).
					Return(nil)

				mockClient.EXPECT().
					AddAlias(gomock.Any(), gomock.Any(), "production").
					Return(nil)

				err := verifyAdapter.Promote(ctx,
					mustParse("ghcr.io/org/components//github.com/org/app:latest"),
					mustParse("ghcr.io/org/components//github.com/org/app:production"),
				)

				Expect(err).ToNot(HaveOccurred())
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
					mustParse("ghcr.io/org/components//github.com/org/app:1.0.0"),
					mustParse("ghcr.io/org/components//github.com/org/app:production"),
				)

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
					mustParse("ghcr.io/org/source-repo//github.com/org/app:1.0.0"),
					mustParse("ghcr.io/org/target-repo//github.com/org/app:production"),
				)

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
					mustParse("ghcr.io/org/source-repo//github.com/org/app:1.0.0"),
					mustParse("ghcr.io/org/target-repo//github.com/org/app:production"),
				)

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
					mustParse("ghcr.io/org/components//github.com/org/app:1.0.0"),
					mustParse("ghcr.io/org/components//github.com/org/app:production"),
				)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to add alias"))
			})
		})

		Context("when source uses an alias that resolves to a version", func() {
			It("promotes the resolved version", func() {
				mockClient.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, ref compref.Ref) (descruntime.Descriptor, error) {
						Expect(ref.Version).To(Equal("latest"))
						return makeDescriptor("2.5.3"), nil
					})

				mockClient.EXPECT().
					AddAlias(gomock.Any(), gomock.Any(), "production").
					DoAndReturn(func(_ context.Context, ref compref.Ref, _ string) error {
						Expect(ref.Version).To(Equal("2.5.3"))
						return nil
					})

				err := adapter.Promote(ctx,
					mustParse("ghcr.io/org/components//github.com/org/app:latest"),
					mustParse("ghcr.io/org/components//github.com/org/app:production"),
				)

				Expect(err).ToNot(HaveOccurred())
			})
		})
	})

	Describe("PromotionPortProvider", func() {
		It("returns a valid OcmPromotionPort", func() {
			port := NewPromotionPortProvider().NewOcmPromotionPort(mockClient)

			Expect(port).ToNot(BeNil())
			Expect(port).To(BeAssignableToTypeOf(&PromotionAdapter{}))
		})
	})
})

var _ = Describe("sameLocation", func() {
	DescribeTable("compares OCI repositories",
		func(a, b compref.Ref, expected bool) {
			result, err := sameLocation(a, b)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(expected))
		},
		Entry("same base url and sub path",
			mustParse("ghcr.io/org/components//github.com/org/app:1.0.0"),
			mustParse("ghcr.io/org/components//github.com/org/app:2.0.0"),
			true,
		),
		Entry("different base url",
			mustParse("ghcr.io/org-a/components//github.com/org/app:1.0.0"),
			mustParse("ghcr.io/org-b/components//github.com/org/app:1.0.0"),
			false,
		),
		Entry("different sub path",
			mustParse("ghcr.io/org/source-repo//github.com/org/app:1.0.0"),
			mustParse("ghcr.io/org/target-repo//github.com/org/app:1.0.0"),
			false,
		),
	)

	Context("with non-OCI repositories", func() {
		It("returns error when source is not OCI", func() {
			a := compref.Ref{Repository: &nonOCIRepository{}}
			b := compref.Ref{Repository: &ocispec.Repository{}}

			_, err := sameLocation(a, b)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("source repository is not an OCI repository"))
		})

		It("returns error when target is not OCI", func() {
			a := compref.Ref{Repository: &ocispec.Repository{}}
			b := compref.Ref{Repository: &nonOCIRepository{}}

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

func mustParse(ref string) compref.Ref {
	r, err := konfcompref.Parse(ref)
	ExpectWithOffset(1, err).ToNot(HaveOccurred())
	return *r
}

type nonOCIRepository struct{}

func (n *nonOCIRepository) GetType() runtime.Type        { return runtime.Type{Name: "other"} }
func (n *nonOCIRepository) SetType(_ runtime.Type)       {}
func (n *nonOCIRepository) DeepCopyTyped() runtime.Typed { return &nonOCIRepository{} }

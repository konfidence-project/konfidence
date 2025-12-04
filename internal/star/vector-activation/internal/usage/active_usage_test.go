package usage

import (
	"context"

	common "github.com/konfidence-project/crds/api/common/v1alpha1"
	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"
	. "github.com/konfidence-project/landscape-vector-activation-controller/test/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("active usage tests", func() {
	var (
		ctx        context.Context
		mockCtrl   *gomock.Controller
		clientMock *MockClient
		stage      *common.Stage
		scheme     *runtime.Scheme
	)
	BeforeEach(func() {
		ctx = context.Background()
		mockCtrl = gomock.NewController(GinkgoT())
		clientMock = NewMockClient(mockCtrl)
		scheme = runtime.NewScheme()
		_ = landscape.AddToScheme(scheme)
		_ = common.AddToScheme(scheme)

		stage = &common.Stage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "stage-test",
				Namespace: "default",
				UID:       "12345",
			},
		}
	})

	Context("Active Usage", func() {
		It("should return usage and no error", func() {
			name := "test-usage"
			namespace := "default"

			clientMock.EXPECT().Get(ctx, gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, namespacedName types.NamespacedName, obj interface{}, _ ...interface{}) error {
					usage := obj.(*landscape.StageVersionUsage)
					usage.Name = name
					usage.Namespace = namespace
					return nil
				})

			usage, err := GetCurrentActiveUsage(ctx, clientMock, stage)
			Expect(err).ToNot(HaveOccurred())
			Expect(usage).ToNot(BeNil())
			Expect(usage.Name).To(Equal(name))

			clientMock.EXPECT().Get(ctx, gomock.Any(), gomock.Any()).
				Return(apierrors.NewNotFound(schema.GroupResource{Group: "landscape.konfidence.io", Resource: "stageversionusages"}, "not-found"))

			usage, err = GetCurrentActiveUsage(ctx, clientMock, stage)
			Expect(err).ToNot(HaveOccurred())
			Expect(usage).To(BeNil())
		})

		It("should create active usage", func() {
			clientMock.EXPECT().Create(ctx, gomock.Any()).Return(nil)
			clientMock.EXPECT().Scheme().Return(scheme)
			newUsage, err := CreateActiveUsage(ctx, clientMock, stage, &landscape.StageVersion{})

			Expect(err).ToNot(HaveOccurred())
			Expect(newUsage).ToNot(BeNil())
			Expect(newUsage.Labels[ActiveStageVersion]).To(Equal(stage.Name))
			Expect(newUsage.Spec.Reason).To(Equal(StageVersionUsageActiveType))
		})

		It("should update active usage", func() {
			clientMock.EXPECT().Patch(ctx, gomock.Any(), gomock.Any()).Return(nil)
			clientMock.EXPECT().Scheme().Return(scheme)
			usage := &landscape.StageVersionUsage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "usage-to-update",
					Namespace: "default",
				},
			}
			err := UpdateActiveUsage(ctx, clientMock, stage, usage)
			Expect(err).ToNot(HaveOccurred())
			Expect(usage.OwnerReferences).ToNot(BeEmpty())
		})

		It("should create or update active usage", func() {
			clientMock.EXPECT().Create(ctx, gomock.Any()).Return(nil)
			clientMock.EXPECT().Scheme().Return(scheme)

			err := CreateOrUpdateActiveUsage(ctx, clientMock, nil, stage, &landscape.StageVersion{})

			Expect(err).ToNot(HaveOccurred())

			clientMock.EXPECT().Patch(ctx, gomock.Any(), gomock.Any()).Return(nil)
			clientMock.EXPECT().Scheme().Return(scheme)

			usage := &landscape.StageVersionUsage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "usage-to-update",
					Namespace: "default",
				},
			}
			err = CreateOrUpdateActiveUsage(ctx, clientMock, usage, stage, &landscape.StageVersion{})

			Expect(err).ToNot(HaveOccurred())
		})

		It("is newer than current active usage", func() {
			stageVersion := &landscape.StageVersion{
				Spec: landscape.StageVersionSpec{
					StageGeneration: int64(3),
				},
			}
			activeUsage := &landscape.StageVersionUsage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "active-usage",
					Namespace: "default",
				},
				Spec: landscape.StageVersionUsageSpec{
					StageVersionRef: &landscape.StageVersionReference{Name: "active-stage-version"},
				},
			}
			activeStageVersion := &landscape.StageVersion{Spec: landscape.StageVersionSpec{
				StageGeneration: int64(2),
			}}
			clientMock.EXPECT().Get(ctx, gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ types.NamespacedName, obj interface{}, _ ...interface{}) error {
					stagerVersion := obj.(*landscape.StageVersion)
					*stagerVersion = *activeStageVersion
					return nil
				})

			result, err := IsNewerThanCurrentActiveUsage(ctx, clientMock, stageVersion, activeUsage)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeTrue())
		})
	})
})

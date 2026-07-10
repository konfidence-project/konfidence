package usage

import (
	"context"

	star "github.com/konfidence-project/konfidence/api/star/v1alpha1"
	. "github.com/konfidence-project/konfidence/internal/vectoractivation/internal/usage/mocks"
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
		ctx          context.Context
		mockCtrl     *gomock.Controller
		clientMock   *MockClient
		stage        *star.Stage
		stageVersion *star.StageVersion
		scheme       *runtime.Scheme
	)
	BeforeEach(func() {
		ctx = context.Background()
		mockCtrl = gomock.NewController(GinkgoT())
		clientMock = NewMockClient(mockCtrl)
		scheme = runtime.NewScheme()
		_ = star.AddToScheme(scheme)

		stage = &star.Stage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "stage-test",
				Namespace: "default",
				UID:       "12345",
			},
		}
		stageVersion = &star.StageVersion{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "stage-version-test",
				Namespace: "default",
			},
		}
	})

	Context("Active Usage", func() {
		It("should return usage and no error", func() {
			name := "test-usage"
			namespace := "default"

			clientMock.EXPECT().Get(ctx, gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, namespacedName types.NamespacedName, obj any, _ ...any) error {
					usage := obj.(*star.StageVersionUsage)
					usage.Name = name
					usage.Namespace = namespace
					return nil
				})

			usage, err := GetCurrentActiveUsage(ctx, clientMock, stage)
			Expect(err).ToNot(HaveOccurred())
			Expect(usage).ToNot(BeNil())
			Expect(usage.Name).To(Equal(name))

			clientMock.EXPECT().Get(ctx, gomock.Any(), gomock.Any()).
				Return(apierrors.NewNotFound(schema.GroupResource{Group: "star.konfidence.io", Resource: "stageversionusages"}, "not-found"))

			usage, err = GetCurrentActiveUsage(ctx, clientMock, stage)
			Expect(err).ToNot(HaveOccurred())
			Expect(usage).To(BeNil())
		})

		It("should create active usage", func() {
			clientMock.EXPECT().Create(ctx, gomock.Any()).Return(nil)
			clientMock.EXPECT().Scheme().Return(scheme)
			newUsage, err := CreateActiveUsage(ctx, clientMock, stage, &star.StageVersion{})

			Expect(err).ToNot(HaveOccurred())
			Expect(newUsage).ToNot(BeNil())
			Expect(newUsage.Labels[ActiveStageVersion]).To(Equal(stage.Name))
			Expect(newUsage.Spec.Reason).To(Equal(StageVersionUsageActiveType))
		})

		It("should update active usage", func() {
			clientMock.EXPECT().Update(ctx, gomock.Any()).Return(nil)
			usage := &star.StageVersionUsage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "usage-to-update",
					Namespace: "default",
				},
			}
			err := UpdateActiveUsage(ctx, clientMock, usage, stageVersion)
			Expect(err).ToNot(HaveOccurred())
			Expect(usage.Spec.StageVersionRef).ToNot(BeNil())
		})

		It("should create or update active usage", func() {
			clientMock.EXPECT().Create(ctx, gomock.Any()).Return(nil)
			clientMock.EXPECT().Scheme().Return(scheme)

			err := CreateOrUpdateActiveUsage(ctx, clientMock, nil, stage, &star.StageVersion{})

			Expect(err).ToNot(HaveOccurred())

			clientMock.EXPECT().Update(ctx, gomock.Any()).Return(nil)

			usage := &star.StageVersionUsage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "usage-to-update",
					Namespace: "default",
				},
			}
			err = CreateOrUpdateActiveUsage(ctx, clientMock, usage, stage, stageVersion)

			Expect(err).ToNot(HaveOccurred())
			Expect(usage.Spec.StageVersionRef).ToNot(BeNil())
		})

		It("is newer than current active usage", func() {
			stageVersion := &star.StageVersion{
				Spec: star.StageVersionSpec{
					StageGeneration: int64(3),
				},
			}
			activeUsage := &star.StageVersionUsage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "active-usage",
					Namespace: "default",
				},
				Spec: star.StageVersionUsageSpec{
					StageVersionRef: &star.StageVersionReference{Name: "active-stage-version"},
				},
			}
			activeStageVersion := &star.StageVersion{Spec: star.StageVersionSpec{
				StageGeneration: int64(2),
			}}
			clientMock.EXPECT().Get(ctx, gomock.Any(), gomock.Any()).DoAndReturn(
				func(_ context.Context, _ types.NamespacedName, obj any, _ ...any) error {
					stagerVersion := obj.(*star.StageVersion)
					*stagerVersion = *activeStageVersion
					return nil
				})

			result, err := IsNewerThanCurrentActiveUsage(ctx, clientMock, stageVersion, activeUsage)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeTrue())
		})
	})
})

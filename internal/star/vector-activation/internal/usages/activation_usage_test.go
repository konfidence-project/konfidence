package usages

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
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var _ = Describe("active usage tests", func() {
	var (
		ctx          context.Context
		mockCtrl     *gomock.Controller
		clientMock   *MockClient
		stage        *common.Stage
		stageVersion *landscape.StageVersion
		activation   *landscape.VectorActivation
	)
	BeforeEach(func() {
		ctx = context.Background()
		mockCtrl = gomock.NewController(GinkgoT())
		clientMock = NewMockClient(mockCtrl)
		stage = &common.Stage{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "stage-test",
				Namespace: "default",
				UID:       "12345",
			},
		}
		stageVersion = &landscape.StageVersion{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "stage-version-test",
				Namespace: "default",
			},
		}
		activation = &landscape.VectorActivation{}
	})

	Context("Activation Usage", func() {
		It("should create activation usage and no error", func() {
			clientMock.EXPECT().Create(ctx, gomock.Any()).Return(nil)
			expectedName := "stage-version-test-activation-usage"

			usage, err := CreateOrUpdateActivationUsage(ctx, clientMock, stage, stageVersion, activation)

			Expect(err).ToNot(HaveOccurred())
			Expect(usage).NotTo(BeNil())
			Expect(usage.Name).To(Equal(expectedName))
			Expect(usage.Labels[ActivationStageVersionUsage]).To(Equal(stage.Name))
			Expect(usage.Spec.Reason).To(Equal(StageVersionUsageActivationType))
			Expect(usage.OwnerReferences).To(HaveLen(1))
		})

		It("should update activation usage if it already exists", func() {
			clientMock.EXPECT().Create(ctx, gomock.Any()).Return(
				apierrors.NewAlreadyExists(schema.GroupResource{Group: "landscape.konfidence.io", Resource: "stageversionusages"}, "stage-version-test-activation-usage"),
			)
			clientMock.EXPECT().Update(ctx, gomock.Any()).Return(nil)

			usage, err := CreateOrUpdateActivationUsage(ctx, clientMock, stage, stageVersion, activation)
			Expect(err).ToNot(HaveOccurred())
			Expect(usage).NotTo(BeNil())

		})

		It("should delete", func() {
			clientMock.EXPECT().Delete(ctx, gomock.Any()).Return(nil)

			err := DeleteActivationUsage(ctx, clientMock, &landscape.StageVersionUsage{})
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

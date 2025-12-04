package usage

import (
	"context"

	common "github.com/konfidence-project/crds/api/common/v1alpha1"
	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"
	. "github.com/konfidence-project/landscape-vector-activation-controller/test/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("active usage tests", func() {
	var (
		ctx        context.Context
		mockCtrl   *gomock.Controller
		clientMock *MockClient
		stage      *common.Stage
		activation *landscape.VectorActivation
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
		activation = &landscape.VectorActivation{
			Spec: landscape.VectorActivationSpec{
				StageVersion: "stage-version-test",
			},
		}
	})

	Context("Activation Usage", func() {
		It("should create activation usage and no error", func() {
			clientMock.EXPECT().Create(ctx, gomock.Any()).Return(nil)

			usage, err := CreateActivationUsage(ctx, clientMock, stage, activation)

			Expect(err).ToNot(HaveOccurred())
			Expect(usage).NotTo(BeNil())
			Expect(usage.Name).To(ContainSubstring("activation-"))
			Expect(usage.Labels[ActivationStageVersionUsage]).To(Equal(stage.Name))
			Expect(usage.Spec.Reason).To(Equal(StageVersionUsageActivationType))
			Expect(usage.Spec.StageVersionRef.Name).To(Equal(activation.Spec.StageVersion))
			Expect(usage.OwnerReferences).To(HaveLen(1))
		})

		It("should delete", func() {
			clientMock.EXPECT().Delete(ctx, gomock.Any()).Return(nil)

			err := DeleteActivationUsage(ctx, clientMock, &landscape.StageVersionUsage{})
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

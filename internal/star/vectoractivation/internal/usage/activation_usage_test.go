package usage

import (
	"context"

	star "github.com/konfidence-project/konfidence/api/star/v1alpha1"
	. "github.com/konfidence-project/konfidence/internal/star/vectoractivation/internal/usage/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var _ = Describe("active usage tests", func() {
	var (
		ctx        context.Context
		mockCtrl   *gomock.Controller
		clientMock *MockClient
		stage      *star.Stage
		activation *star.VectorActivation
		scheme     *runtime.Scheme
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
		activation = &star.VectorActivation{
			ObjectMeta: metav1.ObjectMeta{
				Name: "123",
			},
			Spec: star.VectorActivationSpec{
				StageVersion: "stage-version-test",
			},
		}
	})

	Context("Activation Usage", func() {
		It("should create activation usage and no error", func() {
			clientMock.EXPECT().Scheme().Return(scheme)
			clientMock.EXPECT().Create(ctx, gomock.Any()).Return(nil)

			usage, err := CreateActivationUsage(ctx, clientMock, stage, activation)

			Expect(err).ToNot(HaveOccurred())
			Expect(usage).NotTo(BeNil())
			Expect(usage.Name).To(Equal("stage-test-123-activation"))
			Expect(usage.Labels[ActivationStageVersionUsage]).To(Equal(stage.Name))
			Expect(usage.Spec.Reason).To(Equal(StageVersionUsageActivationType))
			Expect(usage.Spec.StageVersionRef.Name).To(Equal(activation.Spec.StageVersion))
			Expect(usage.OwnerReferences).To(HaveLen(1))
		})

		It("should delete", func() {
			clientMock.EXPECT().Delete(ctx, gomock.Any()).Return(nil)

			err := DeleteActivationUsage(ctx, clientMock, &star.StageVersionUsage{})
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

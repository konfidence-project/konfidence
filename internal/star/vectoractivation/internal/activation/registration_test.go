package activation

import (
	"context"

	star "github.com/konfidence-project/konfidence/api/star/v1alpha1"
	. "github.com/konfidence-project/konfidence/internal/star/vectoractivation/internal/activation/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("registration tests", func() {
	var (
		ctx        context.Context
		mockCtrl   *gomock.Controller
		clientMock *MockClient
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockCtrl = gomock.NewController(GinkgoT())
		clientMock = NewMockClient(mockCtrl)
	})

	Context("GetRegistrations", func() {
		It("should return registration list and no error", func() {
			namespace := "default"
			clientMock.EXPECT().List(ctx, gomock.Any(), client.InNamespace(namespace)).
				DoAndReturn(func(_ context.Context, list interface{}, _ ...interface{}) error {
					regList := list.(*star.ActivationTaskRegistrationList)
					regList.Items = append(regList.Items, star.ActivationTaskRegistration{})
					return nil
				})

			result, err := GetRegistrations(ctx, clientMock, namespace)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeNil())
			Expect(result.Items).To(HaveLen(1))
		})
	})
})

package activation

import (
	"context"

	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"
	. "github.com/konfidence-project/landscape-vector-activation-controller/test/mocks"
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
					regList := list.(*landscape.ActivationTaskRegistrationList)
					regList.Items = append(regList.Items, landscape.ActivationTaskRegistration{})
					return nil
				})

			result, err := GetRegistrations(ctx, clientMock, namespace)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeNil())
			Expect(result.Items).To(HaveLen(1))
		})
	})
})

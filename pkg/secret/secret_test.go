package secret

import (
	"context"

	. "github.com/konfidence-project/konfidence/pkg/secret/internal/mocks"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("pkg auth functions", func() {
	var (
		clientMock *MockClient
		mockCtrl   *gomock.Controller
		ctx        context.Context
	)

	const (
		KonfidenceSystemNamespace = "konfidence-system"
		ConfigMapName             = "auth-configuration"
		AuthConfigMapKey          = "authenticationSecretRefs"
		SecretName                = "dockerio-secret"
		HostName                  = "test.registry.com"
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		clientMock = NewMockClient(mockCtrl)
		ctx = context.Background()
	})

	AfterEach(func() {

	})

	Context("When resolving secret name", func() {
		It("should successfully extract secret from ConfigMap", func() {
			configMap := &corev1.ConfigMap{Data: map[string]string{
				AuthConfigMapKey: HostName + ": " + SecretName,
			}}
			clientMock.EXPECT().Get(ctx, types.NamespacedName{
				Namespace: KonfidenceSystemNamespace,
				Name:      ConfigMapName,
			}, gomock.Any()).DoAndReturn(
				func(_ context.Context, _ types.NamespacedName, obj interface{}, _ ...interface{}) error {
					*obj.(*corev1.ConfigMap) = *configMap
					return nil
				})

			secretName, err := GetSecretByConfigMap(ctx, clientMock, ConfigMapName, HostName)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(secretName).To(gomega.Equal(SecretName))
		})
	})
})

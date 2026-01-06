package ocm_test

import (
	"context"
	"fmt"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/konfidence-project/landscape-vector-deployment-controller/pkg/ocm"
	. "github.com/konfidence-project/landscape-vector-deployment-controller/test/mocks"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
)

var _ = Describe("ocm context provider functions", func() {
	var (
		clientMock *MockClient
		mockCtrl   *gomock.Controller
		ctx        context.Context
	)

	const (
		RegistryUrl                  = "https://docker.io/ocm"
		KonfidenceSystemNamespace    = "konfidence-system"
		Namespace                    = "test"
		DefaultConfigMapName         = "vector-deployment-controller-configuration"
		AuthConfigMapKey             = "authenticationSecretRefs"
		SecretName                   = "dockerio-secret"
		ConventionalNamingSecretName = "docker.io"
		HostName                     = "docker.io"
		User                         = "user"
		Password                     = "ghp_pass"
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		clientMock = NewMockClient(mockCtrl)
		ctx = context.Background()
	})

	AfterEach(func() {

	})

	Context("When resolving authentication credentials for OCM context", func() {
		It("should successfully extract credentials from ConfigMap", func() {
			configMap := &v1.ConfigMap{Data: map[string]string{
				AuthConfigMapKey: HostName + ": " + SecretName,
			}}
			clientMock.EXPECT().Get(ctx, types.NamespacedName{
				Namespace: KonfidenceSystemNamespace,
				Name:      DefaultConfigMapName,
			}, gomock.Any()).DoAndReturn(
				func(_ context.Context, _ types.NamespacedName, obj interface{}, _ ...interface{}) error {
					*obj.(*v1.ConfigMap) = *configMap
					return nil
				})

			secret := &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      SecretName,
					Namespace: Namespace,
				},
				Type: v1.SecretTypeDockerConfigJson,
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte("{\"auths\":{\"" + HostName + "\":{\"username\":\"" + User + "\",\"password\":\"" +
						Password + "\",\"auth\":\"dXNlcjpnaHBfcGFzcw==\"}}}"),
				},
			}
			clientMock.EXPECT().Get(ctx, types.NamespacedName{
				Namespace: Namespace,
				Name:      SecretName,
			}, gomock.Any()).DoAndReturn(
				func(_ context.Context, _ types.NamespacedName, obj interface{}, _ ...interface{}) error {
					*obj.(*v1.Secret) = *secret
					return nil
				})

			secret, err := ocm.NewOCMContextProvider(clientMock).GetCredentials(ctx, Namespace, RegistryUrl)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(secret.Name).To(gomega.Equal(SecretName))
			gomega.Expect(secret.Namespace).To(gomega.Equal(Namespace))
			user, pwd, _ := extractCredentials(HostName, secret)
			gomega.Expect(user).To(gomega.Equal(User))
			gomega.Expect(pwd).To(gomega.Equal(Password))
		})
		It("should successfully extract credentials directly per secret by naming convention", func() {
			clientMock.EXPECT().Get(ctx, types.NamespacedName{
				Namespace: KonfidenceSystemNamespace,
				Name:      DefaultConfigMapName,
			}, gomock.Any()).DoAndReturn(
				func(_ context.Context, _ types.NamespacedName, obj interface{}, _ ...interface{}) error {
					return errors.NewNotFound(
						schema.GroupResource{
							Group:    "",
							Resource: "configmaps",
						},
						DefaultConfigMapName,
					)
				})

			secret := &v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      HostName,
					Namespace: Namespace,
				},
				Type: v1.SecretTypeDockerConfigJson,
				Data: map[string][]byte{
					v1.DockerConfigJsonKey: []byte("{\"auths\":{\"" + HostName + "\":{\"username\":\"" + User + "\",\"password\":\"" +
						Password + "\",\"auth\":\"dXNlcjpnaHBfcGFzcw==\"}}}"),
				},
			}
			clientMock.EXPECT().Get(ctx, types.NamespacedName{
				Namespace: Namespace,
				Name:      ConventionalNamingSecretName,
			}, gomock.Any()).DoAndReturn(
				func(_ context.Context, _ types.NamespacedName, obj interface{}, _ ...interface{}) error {
					*obj.(*v1.Secret) = *secret
					return nil
				})

			secret, err := ocm.NewOCMContextProvider(clientMock).GetCredentials(ctx, Namespace, RegistryUrl)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(secret.Name).To(gomega.Equal(HostName))
			gomega.Expect(secret.Namespace).To(gomega.Equal(Namespace))
			user, pwd, _ := extractCredentials(HostName, secret)
			gomega.Expect(user).To(gomega.Equal(User))
			gomega.Expect(pwd).To(gomega.Equal(Password))
		})
		It("should successfully provide OCM context without authentication credentials", func() {
			clientMock.EXPECT().Get(ctx, types.NamespacedName{
				Namespace: KonfidenceSystemNamespace,
				Name:      DefaultConfigMapName,
			}, gomock.Any()).DoAndReturn(
				func(_ context.Context, _ types.NamespacedName, obj interface{}, _ ...interface{}) error {
					return errors.NewNotFound(
						schema.GroupResource{
							Group:    "",
							Resource: "configmaps",
						},
						DefaultConfigMapName,
					)
				})

			clientMock.EXPECT().Get(ctx, types.NamespacedName{
				Namespace: Namespace,
				Name:      ConventionalNamingSecretName,
			}, gomock.Any()).DoAndReturn(
				func(_ context.Context, _ types.NamespacedName, obj interface{}, _ ...interface{}) error {
					return errors.NewNotFound(
						schema.GroupResource{
							Group:    "",
							Resource: "secrets",
						},
						ConventionalNamingSecretName,
					)
				})

			secret, err := ocm.NewOCMContextProvider(clientMock).GetCredentials(ctx, Namespace, RegistryUrl)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(secret).To(gomega.BeNil())
		})
		It("should provide OCM context without authentication credentials when registry url is empty", func() {
			secret, err := ocm.NewOCMContextProvider(clientMock).GetCredentials(ctx, Namespace, "")
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(secret).To(gomega.BeNil())
		})
	})
})

func extractCredentials(domain string, s *v1.Secret) (string, string, error) {
	dockerConfigJson, ok := s.Data[v1.DockerConfigJsonKey]
	if !ok {
		return "", "", fmt.Errorf("could not parse secret")
	}

	var dockerConfig configfile.ConfigFile
	if err := json.Unmarshal(dockerConfigJson, &dockerConfig); err != nil {
		return "", "", fmt.Errorf("failed to unmarshal secret")
	}

	authConfig, ok := dockerConfig.AuthConfigs[domain]
	if !ok {
		return "", "", fmt.Errorf("could not get authConfig")
	}

	return authConfig.Username, authConfig.Password, nil
}

package config_test

import (
	"github.com/konfidence-project/konfidence/internal/star/galaxysync/internal/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// minimalKubeconfig is a syntactically valid kubeconfig that points to a
// placeholder server - sufficient for clientcmd.RESTConfigFromKubeConfig.
const minimalKubeconfig = `apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://remote-cluster.example.com
  name: remote
contexts:
- context:
    cluster: remote
    user: remote-user
  name: remote-ctx
current-context: remote-ctx
users:
- name: remote-user
  user: {}
`

var _ = Describe("FromSecret", func() {
	const namespace = "konfidence-system"

	Describe("when the Secret exists and contains a valid kubeconfig", func() {
		It("should return a rest.Config with the correct host and credentials", func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      config.SecretName,
					Namespace: namespace,
				},
				Data: map[string][]byte{
					config.SecretKey: []byte(minimalKubeconfig),
				},
			}

			c := fake.NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(secret).
				Build()

			cfg, err := config.FromSecret(c, namespace)

			Expect(err).NotTo(HaveOccurred())
			Expect(cfg).NotTo(BeNil())
			Expect(cfg.Host).To(Equal("https://remote-cluster.example.com"))
		})
	})

	Describe("when the Secret does not exist", func() {
		It("should return nil to signal single-cluster fallback", func() {
			c := fake.NewClientBuilder().
				WithScheme(scheme.Scheme).
				Build()

			cfg, err := config.FromSecret(c, namespace)

			Expect(err).NotTo(HaveOccurred())
			Expect(cfg).To(BeNil())
		})
	})
})

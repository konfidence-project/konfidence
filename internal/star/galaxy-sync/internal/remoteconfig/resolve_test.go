/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package remoteconfig_test

import (
	"github.com/konfidence-project/landscape-gcp-sync-controller/internal/remoteconfig"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// minimalKubeconfig is a syntactically valid kubeconfig that points to a
// placeholder server — sufficient for clientcmd.RESTConfigFromKubeConfig.
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
					Name:      remoteconfig.SecretName,
					Namespace: namespace,
				},
				Data: map[string][]byte{
					remoteconfig.SecretKey: []byte(minimalKubeconfig),
				},
			}

			c := fake.NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(secret).
				Build()

			cfg, err := remoteconfig.FromSecret(c, namespace)

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

			cfg, err := remoteconfig.FromSecret(c, namespace)

			Expect(err).NotTo(HaveOccurred())
			Expect(cfg).To(BeNil())
		})
	})
})

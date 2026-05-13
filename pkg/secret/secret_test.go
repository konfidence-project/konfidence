/*
Copyright 2025.

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

package secret

import (
	"context"

	. "github.com/konfidence-project/pkg/secret/internal/mocks"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	v1 "k8s.io/api/core/v1"
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
			configMap := &v1.ConfigMap{Data: map[string]string{
				AuthConfigMapKey: HostName + ": " + SecretName,
			}}
			clientMock.EXPECT().Get(ctx, types.NamespacedName{
				Namespace: KonfidenceSystemNamespace,
				Name:      ConfigMapName,
			}, gomock.Any()).DoAndReturn(
				func(_ context.Context, _ types.NamespacedName, obj interface{}, _ ...interface{}) error {
					*obj.(*v1.ConfigMap) = *configMap
					return nil
				})

			secretName, err := GetSecretByConfigMap(ctx, clientMock, ConfigMapName, HostName)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(secretName).To(gomega.Equal(SecretName))
		})
	})
})

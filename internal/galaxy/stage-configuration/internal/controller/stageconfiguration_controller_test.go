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

package controller

import (
	"context"
	"time"

	common "github.com/konfidence-project/crds/api/common/v1alpha1"
	global "github.com/konfidence-project/crds/api/global/v1alpha1"
	"github.com/konfidence-project/gcp-stage-configuration-controller/ocm/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("StageConfiguration Controller", func() {
	const (
		Vector           = "http://localhost:5100//konfidence.cloud/project/constructed-vector"
		DefaultNameSpace = "default"
		timeout          = time.Second * 10
		interval         = time.Millisecond * 250
	)

	var (
		reconciler    *StageConfigurationReconciler
		ocmClientMock *mocks.MockClient
		mockCtrl      *gomock.Controller
	)

	BeforeEach(func() {
		// mock setup
		mockCtrl = gomock.NewController(GinkgoT())
		ocmClientMock = mocks.NewMockClient(mockCtrl)

		reconciler = &StageConfigurationReconciler{
			Client:    k8sManager.GetClient(),
			Scheme:    k8sManager.GetScheme(),
			OCMClient: ocmClientMock,
		}
		err := reconciler.SetupWithManager(k8sManager)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		ctx := context.Background()
		managerClient := k8sManager.GetClient()

		// Cleanup Resources
		err := managerClient.DeleteAllOf(ctx, &global.StageConfiguration{}, client.InNamespace(DefaultNameSpace))
		Expect(err).ToNot(HaveOccurred())

		err = managerClient.DeleteAllOf(ctx, &common.Stage{}, client.InNamespace(DefaultNameSpace))
		Expect(err).ToNot(HaveOccurred())

		if mockCtrl != nil {
			mockCtrl.Finish()
		}
	})

	Context("When reconciling a stageConfiguration", func() {
		It("should successfully create a stage with latest vector version ", func() {
		
		})
		It("should successfully update an existing stage with latest vector version ", func() {

		})
	})
})

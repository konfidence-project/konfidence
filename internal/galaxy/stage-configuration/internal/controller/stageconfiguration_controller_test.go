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
	testutil "github.com/konfidence-project/gcp-stage-configuration-controller/internal/utils"
	"github.com/konfidence-project/gcp-stage-configuration-controller/pkg/ocm/mocks"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Stage Configuration Controller", Ordered, func() {
	const (
		StageConfiguration = "stage-configuration-dev"
		StageDev           = "stage-dev"
		V100               = "1.0.0"
		V101               = "1.0.1"
		Vector             = "http://localhost:5100//konfidence.cloud/project/constructed-vector"
		VectorV100         = "http://localhost:5100//konfidence.cloud/project/constructed-vector" + ":" + V100
		VectorV101         = "http://localhost:5100//konfidence.cloud/project/constructed-vector" + ":" + V101
		Namespace          = "default"
		TargetNamespace    = "target"
		timeout            = time.Second * 10
		interval           = time.Millisecond * 250
	)

	var (
		reconciler    *StageConfigurationReconciler
		ocmClientMock *mocks.MockClient
		mockCtrl      *gomock.Controller
	)

	BeforeAll(func() {
		// mock setup
		mockCtrl = gomock.NewController(GinkgoT())
		ocmClientMock = mocks.NewMockClient(mockCtrl)

		reconciler = &StageConfigurationReconciler{
			Mgr:       k8sManager,
			OCMClient: ocmClientMock,
			Scheme:    k8sScheme,
			SkipOci:   false,
		}
		err := reconciler.SetupWithManager(k8sManager)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterAll(func() {
		if mockCtrl != nil {
			mockCtrl.Finish()
		}
	})

	BeforeEach(func() {
		testutil.CleanupResources(context.Background(), k8sClient, Namespace, TargetNamespace)
	})

	AfterEach(func() {
		testutil.CleanupResources(context.Background(), k8sClient, Namespace, TargetNamespace)
	})

	Context("When reconciling a stageConfiguration", func() {
		It("should successfully create a stage with latest vector version ", func() {
			ctx := context.Background()
			ocmClientMock.EXPECT().GetLatestVectorVersion(gomock.Any(), Vector).Return(V100, nil)
			// create target namespace
			// note: since test env cannot delete namespaces the target namespace is created once in the first test
			testutil.CreateNamespace(ctx, k8sClient, TargetNamespace)
			ns := &v1.Namespace{}
			nsLookupKey := client.ObjectKey{
				Name: TargetNamespace,
			}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, nsLookupKey, ns)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			// create stage configuration
			testutil.CreateStageConfiguration(ctx, k8sClient, StageConfiguration, Namespace, TargetNamespace, StageDev, Vector)

			// check that the stage configuration has been created and has valid properties
			stageConfiguration := &global.StageConfiguration{}
			stageConfigurationLookupKey := types.NamespacedName{Name: StageConfiguration, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageConfigurationLookupKey, stageConfiguration)).To(Succeed())
				g.Expect(stageConfiguration.Spec.Name).To(Equal(StageDev))
				g.Expect(stageConfiguration.Spec.Vector).To(Equal(Vector))
				g.Expect(stageConfiguration.Spec.TargetNamespace).To(Equal(TargetNamespace))
			}, timeout, interval).Should(Succeed())

			// check that the stage has been created and has valid properties
			stage := &common.Stage{}
			stageLookupKey := types.NamespacedName{Name: StageDev, Namespace: TargetNamespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
				g.Expect(stage.Name).To(Equal(StageDev))
				g.Expect(stage.Spec.Vector).To(Equal(VectorV100))
			}, timeout, interval).Should(Succeed())
		})
		It("should successfully update an existing stage with latest vector version ", func() {
			ctx := context.Background()
			ocmClientMock.EXPECT().GetLatestVectorVersion(gomock.Any(), Vector).Return(V101, nil)

			// create stage with v1.0.0 vector version
			testutil.CreateStage(ctx, k8sClient, StageDev, TargetNamespace, VectorV100)

			// check that the stage has been created and has valid properties
			stage := &common.Stage{}
			stageLookupKey := types.NamespacedName{Name: StageDev, Namespace: TargetNamespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
				g.Expect(stage.Name).To(Equal(StageDev))
				g.Expect(stage.Spec.Vector).To(Equal(VectorV100))
			}, timeout, interval).Should(Succeed())

			// create stage configuration
			testutil.CreateStageConfiguration(ctx, k8sClient, StageConfiguration, Namespace, TargetNamespace, StageDev, Vector)

			// check that the stage configuration has been created and has valid properties
			stageConfiguration := &global.StageConfiguration{}
			stageConfigurationLookupKey := types.NamespacedName{Name: StageConfiguration, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageConfigurationLookupKey, stageConfiguration)).To(Succeed())
				g.Expect(stageConfiguration.Spec.Name).To(Equal(StageDev))
				g.Expect(stageConfiguration.Spec.Vector).To(Equal(Vector))
				g.Expect(stageConfiguration.Spec.TargetNamespace).To(Equal(TargetNamespace))
			}, timeout, interval).Should(Succeed())

			// check that the stage has been updated with new vector version
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
				g.Expect(stage.Spec.Vector).To(Equal(VectorV101))
			}, timeout, interval).Should(Succeed())
		})
	})
})

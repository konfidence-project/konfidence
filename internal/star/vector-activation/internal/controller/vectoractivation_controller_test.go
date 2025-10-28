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

package controller

import (
	"context"
	"time"

	common "github.com/konfidence-project/crds/api/common/v1alpha1"
	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"
	testUtil "github.com/konfidence-project/landscape-vector-activation-controller/internal/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var _ = Describe("VectorActivation Controller", func() {
	const (
		StageDev         = "stage-dev"
		StageVersion     = "stage-version-dev"
		VectorActivation = "stage-version-dev-activation"
		Vector001        = "https://registry.kdenv.lab/ocm/vector//common.konfidence.cloud/example/vector:0.0.1"
		Namespace        = "default"
		timeout          = time.Second * 5
		interval         = time.Millisecond * 250
	)
	BeforeEach(func() {
		testUtil.CleanupVectorActivation(k8sClient, VectorActivation, Namespace)
		testUtil.CleanupStageVersion(k8sClient, StageVersion, Namespace)
		testUtil.CleanupStage(k8sClient, StageDev, Namespace)
	})

	AfterEach(func() {
		testUtil.CleanupVectorActivation(k8sClient, VectorActivation, Namespace)
		testUtil.CleanupStageVersion(k8sClient, StageVersion, Namespace)
		testUtil.CleanupStage(k8sClient, StageDev, Namespace)
	})

	Context("When reconciling a vector activation", func() {

		It("should successfully reconcile the vector activation", func() {
			ctx := context.Background()

			testUtil.CreateStage(ctx, k8sClient, StageDev, Namespace, StageDev, Vector001)

			stage := &common.Stage{}
			stageLookupKey := types.NamespacedName{Name: StageDev, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			testUtil.CreateStageVersion(ctx, k8sClient, StageVersion, Namespace, Vector001)

			// check that the stageVersion has been created and has valid properties
			stageVersion := &landscape.StageVersion{}
			stageVersionLookupKey := types.NamespacedName{Name: StageVersion, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)).To(Succeed())
				g.Expect(stageVersion.Spec.Vector).To(Equal(Vector001))
				g.Expect(stageVersion.Spec.StageGeneration).To(Equal(int64(1)))
			}, timeout, interval).Should(Succeed())

			Expect(controllerutil.SetOwnerReference(stage, stageVersion, k8sClient.Scheme())).To(Succeed())
			testUtil.UpdateStageVersion(ctx, k8sClient, stageVersion)

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageVersionLookupKey, stageVersion)).To(Succeed())
				g.Expect(stageVersion.Spec.Vector).To(Equal(Vector001))
				g.Expect(stageVersion.Spec.StageGeneration).To(Equal(int64(1)))
				g.Expect(stageVersion.GetOwnerReferences()).To(HaveLen(1))
				g.Expect(testUtil.HasOwnerReference(stageVersion.GetOwnerReferences(), metav1.OwnerReference{
					Kind: common.StageKind,
					Name: StageDev,
				})).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			testUtil.CreateVectorActivation(ctx, k8sClient, VectorActivation, Namespace, Vector001, StageVersion)

			// assert vectorActivation properties
			vectorActivation := &landscape.VectorActivation{}
			vectorActivationLookupKey := types.NamespacedName{Name: VectorActivation, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorActivationLookupKey, vectorActivation)).To(Succeed())
				g.Expect(vectorActivation.Spec.StageVersion).To(Equal(StageVersion))
				g.Expect(vectorActivation.Spec.Vector).To(Equal(Vector001))
			}, timeout, interval).Should(Succeed())
		})

	})
})

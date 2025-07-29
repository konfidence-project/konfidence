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

/*
import (
	"context"
	"fmt"
	"time"

	common "github.com/konfidence-project/crds/api/common/v1alpha1"
	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"
	testUtil "github.com/konfidence-project/landscape-stage-controller/internal/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("StageVersion Controller", func() {
	const (
		StageDev         = "stage-dev"
		StageDevSpecName = "dev"
		StageVersion     = "stage-version-dev"
		Namespace        = "default"
		Vector001        = "https://registry.kdenv.lab/ocm/vector//common.konfidence.tools.sap/example/vector:0.0.1"
		VectorName001    = "common.konfidence.tools.sap.example.vector-0.0.1"
		Vector002        = "https://registry.kdenv.lab/ocm/vector//common.konfidence.tools.sap/example/vector:0.0.2"
		VectorName002    = "common.konfidence.tools.sap.example.vector-0.0.2"
		timeout          = time.Second * 10
		interval         = time.Millisecond * 250
	)

	BeforeEach(func() {
		testUtil.CleanupStage(k8sClient, StageDev, Namespace)
		testUtil.CleanupStageVersion(k8sClient, StageVersion, Namespace)
	})

	AfterEach(func() {
		testUtil.CleanupStage(k8sClient, StageDev, Namespace)
		testUtil.CleanupStageVersion(k8sClient, StageVersion, Namespace)
	})

	Context("When reconciling a stageVersion", func() {
		It("should successfully reconcile the stageVersion", func() {
			ctx := context.Background()
			testUtil.CreateStage(ctx, k8sClient, StageDev, Namespace, StageDevSpecName, Vector001)

			// check that the stageVersion has been created and has valid properties
			stageVersion := &landscape.StageVersion{}
			stageVersionLookupKey := types.NamespacedName{Name: StageDev, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
				g.Expect(stage.Spec.Name).To(Equal(StageDevSpecName))
				g.Expect(len(stage.Status.Conditions)).To(Equal(1))
				g.Expect(stage.Status.Conditions[0].Reason).To(Equal(common.VectorDeploymentCreatedCondition))
				g.Expect(stage.Status.Conditions[0].Type).To(Equal(common.VectorDeploymentCreatedCondition))
				g.Expect(stage.Status.Conditions[0].Status).To(Equal(metav1.ConditionTrue))
			}, timeout, interval).Should(Succeed())

			// check that the vectorDeployment has been created and has valid properties
			vectorDeployment := &landscape.VectorDeployment{}
			vectorDeploymentLookupKey := types.NamespacedName{Name: VectorName001, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, vectorDeploymentLookupKey, vectorDeployment)).To(Succeed())
				g.Expect(vectorDeployment.Spec.Vector).To(Equal(stage.Spec.Vector))
				g.Expect(len(vectorDeployment.GetOwnerReferences())).To(Equal(2))
				g.Expect(containsReference(vectorDeployment.GetOwnerReferences(), StageVersion, landscape.StageVersionKind)).To(BeTrue())
				g.Expect(vectorDeployment.Spec.Vector).To(Equal(Vector001))
			}, timeout, interval).Should(Succeed())

			// mark vectorDeployment as deployed
			meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{Type: common.VectorDeployedCondition,
				Status: metav1.ConditionTrue, Reason: landscape.VectorDeployedCondition,
				Message: fmt.Sprintf("Vector has been successfully deployed")})

			Expect(k8sClient.Status().Update(ctx, vectorDeployment)).To(Succeed())

			// check that the stageVersion has status ready
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
				g.Expect(stage.Spec.Name).To(Equal(StageDevSpecName))
				g.Expect(len(stage.Status.Conditions)).To(Equal(2))
				g.Expect(stage.Status.Conditions[1].Reason).To(Equal(common.StageReady))
				g.Expect(stage.Status.Conditions[1].Type).To(Equal(common.StageReady))
				g.Expect(stage.Status.Conditions[1].Status).To(Equal(metav1.ConditionTrue))
			}, timeout, interval).Should(Succeed())

		})

	})
})

*/

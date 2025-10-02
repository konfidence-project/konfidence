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

	landscape "github.com/konfidence-project/crds/api/landscape/v1alpha1"
	testUtil "github.com/konfidence-project/landscape-vector-activation-controller/internal/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("VectorActivation Controller", func() {
	const (
		StageVersion        = "stage-version-dev"
		VectorActivation    = "stage-version-dev-activation"
		ActivationExecution = "activation-execution"
		Vector001           = "https://registry.kdenv.lab/ocm/vector//common.konfidence.cloud/example/vector:0.0.1"
		Execution0          = "stage-version-dev-activation-execution"
		Namespace           = "default"
		timeout             = time.Second * 5
		interval            = time.Millisecond * 250
	)
	BeforeEach(func() {
		testUtil.CleanupVectorActivation(k8sClient, VectorActivation, Namespace)
		testUtil.CleanupStageVersion(k8sClient, StageVersion, Namespace)
		testUtil.CleanupActivationExecution(k8sClient, ActivationExecution, Execution0)
	})

	AfterEach(func() {
		testUtil.CleanupVectorActivation(k8sClient, VectorActivation, Namespace)
		testUtil.CleanupStageVersion(k8sClient, StageVersion, Namespace)
		testUtil.CleanupActivationExecution(k8sClient, ActivationExecution, Execution0)
	})

	Context("When reconciling a vector activation", func() {

		It("should successfully reconcile the vector activation", func() {
			ctx := context.Background()

			testUtil.CreateStageVersion(ctx, k8sClient, StageVersion, Namespace, Vector001)

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

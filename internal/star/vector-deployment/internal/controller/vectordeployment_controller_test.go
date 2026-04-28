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
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	ociv1 "ocm.software/open-component-model/bindings/go/oci/spec/repository/v1/oci"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("VectorDeployment Controller", func() {

	const (
		controllerName  = "vectordeployment"
		vectorReference = "https://registry.kdenv.lab/sample-project//github.com/konfidence-project/sample-vector:0.3.0"
		ocmName         = "common.konfidence.cloud.example.vector-0.3.0"
		testNamespace   = "default"
		artifactPrefix  = "sample-service-1-0-0-1-"
		timeout         = time.Second * 10
		interval        = time.Millisecond * 250
	)

	var (
		reconciler     *VectorDeploymentReconciler
		ocmAdapterMock *MockVectorOcmPort
		mockCtrl       *gomock.Controller
	)

	BeforeEach(func() {
		// Mock setup
		mockCtrl = gomock.NewController(GinkgoT())
		ocmAdapterMock = NewMockVectorOcmPort(mockCtrl)

		// Controller setup
		reconciler = &VectorDeploymentReconciler{
			Client:     k8sManager.GetClient(),
			Scheme:     k8sManager.GetScheme(),
			Recorder:   k8sManager.GetEventRecorder(VectorDeploymentControllerName),
			OcmAdapter: ocmAdapterMock,
		}
		err := reconciler.SetupWithManager(k8sManager, controllerName)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	})

	AfterEach(func() {
		ctx := context.Background()

		managerClient := k8sManager.GetClient()

		// Cleanup VectorDeployments
		err := managerClient.DeleteAllOf(ctx, &landscape.VectorDeployment{}, client.InNamespace(testNamespace))
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		// Cleanup ArtifactDeployments
		err = managerClient.DeleteAllOf(ctx, &landscape.ArtifactDeployment{}, client.InNamespace(testNamespace))
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		// Cleanup VectorAssignments
		err = managerClient.DeleteAllOf(ctx, &landscape.VectorAssignment{}, client.InNamespace(testNamespace))
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		if mockCtrl != nil {
			mockCtrl.Finish()
		}
	})

	It("should successfully reconcile the vector-deployment resource", func() {
		ctx := context.Background()
		// when: GetVectorDescriptor is called on the ocmAdapter mock
		// then: mock should be called with a vector reference and return a VectorDescriptor
		vectorDescriptor := VectorDescriptor{
			References: []compref.Ref{
				{
					Repository: &ociv1.Repository{BaseUrl: "https://registry.kdenv.lab"},
					Component:  "github.com/konfidence-project/sample-service-1",
					Version:    "0.0.1",
				},
			},
			DescriptorJSON: []byte(`{"meta":{"schemaVersion":"v2"},"component":{"name":"github.com/konfidence-project/sample-vector","version":"0.3.0","labels":[{"name":"konfidence-project/sample-vector","value":"01904be8-bae3-ae70-e4d6-78af41d7e0a2","version":"v1"}],"creationTime":"2025-09-22T06:32:45Z","repositoryContexts":null,"provider":"konfidence-project","resources":[],"sources":[],"componentReferences":[{"name":"sample-service-1","version":"0.0.1","componentName":"github.com/konfidence-project/sample-service-1","digest":{"hashAlgorithm":"","normalisationAlgorithm":"","value":""}}]}}`),
		}
		ocmAdapterMock.EXPECT().GetVectorDescriptor(gomock.Any(), gomock.Any()).Return(vectorDescriptor, nil).AnyTimes()

		// when: GetArtifactManifestByReference is called on the ocmAdapter mock
		// then: mock should be called with a vector reference and an artifact reference, and return a ArtifactManifest
		artifactManifest := ArtifactManifest{
			Type:       "cloud.konfidence.flux.helm",
			AllowReuse: true,
			Tasks: []TaskManifest{
				{
					Name:      "sample-service-1-task-1",
					Type:      "k8s-job",
					DependsOn: nil,
					Spec:      "{\"template\":{\"spec\":{\"restartPolicy\":\"Never\",\"containers\":[{\"name\":\"sample-service-1-task-1-container\",\"image\":\"alpine:3.22.1\",\"command\":[\"echo\",\"I am task 1 of service 1\"]}]}},\"backoffLimit\":4}",
				},
				{
					Name:      "sample-service-1-task-2",
					Type:      "k8s-job",
					DependsOn: nil,
					Spec:      "{\"template\":{\"spec\":{\"restartPolicy\":\"Never\",\"containers\":[{\"name\":\"sample-service-1-task-2-container\",\"image\":\"alpine:3.22.1\",\"command\":[\"echo\",\"I am task 2 of service 1\"]}]}},\"backoffLimit\":4}",
				},
			},
			Resources: []OCMResource{
				{
					Name:    "sample-service-1-helm-chart",
					Type:    "helmChart",
					Content: []byte("{\"type\": \"helm\", \"helmChart\": \"podinfo:6.9.1\", \"helmRepository\": \"https://stefanprodan.github.io/podinfo\"}"),
				},
				{
					Name:    "sample-service-1-image",
					Type:    "ociImage",
					Content: []byte("{\"type\": \"ociArtifact\", \"imageReference\": \"stefanprodan/podinfo:6.9.1\"}"),
				},
				{
					Name:    "sample-service-1-task-1",
					Type:    "ociImage",
					Content: []byte("{\"type\": \"ociArtifact\",\"imageReference\": \"alpine:3.22.1\"}"),
				},
				{
					Name:    "sample-service-1-task-2",
					Type:    "ociImage",
					Content: []byte("{\"type\": \"ociArtifact\",\"imageReference\": \"alpine:3.22.1\"}"),
				},
			},
		}
		ocmAdapterMock.EXPECT().GetArtifactManifestByReference(gomock.Any(), gomock.Any()).Return(artifactManifest, nil).AnyTimes()

		// GIVEN: create a VectorDeployment resource
		vectorDeployment := &landscape.VectorDeployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       "VectorDeployment",
				APIVersion: "landscape.konfidence.cloud/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      ocmName,
				Namespace: testNamespace,
				Labels:    map[string]string{"app.kubernetes.io/name": "crds", "app.kubernetes.io/managed-by": "kustomize"},
			},
			Spec: landscape.VectorDeploymentSpec{
				Vector: vectorReference,
			},
			Status: landscape.VectorDeploymentStatus{},
		}

		// WHEN: creating the resource will trigger the reconciler automatically
		err := k8sClient.Create(ctx, vectorDeployment)
		gomega.Expect(err).To(gomega.Succeed())
		By("successfully created VectorDeployment resource")

		// THEN: Verify that the reconciler processed the resource and updated

		By("Verifying ResolvedVectorOcm and VectorDownloaded condition is set")
		actualVectorDeployment := &landscape.VectorDeployment{}
		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: ocmName, Namespace: testNamespace}, actualVectorDeployment)).To(gomega.Succeed())
			g.Expect(actualVectorDeployment.Status.ResolvedVectorOcm).To(gomega.Not(gomega.BeEmpty()))
			g.Expect(meta.IsStatusConditionTrue(actualVectorDeployment.Status.Conditions, landscape.VectorDownloadedCondition)).To(gomega.BeTrue())
		}, timeout, interval).Should(gomega.Succeed())

		By("Verifying ArtifactDeployment was created")
		artifactDeploymentList := &landscape.ArtifactDeploymentList{}
		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(k8sClient.List(ctx, artifactDeploymentList, client.InNamespace(testNamespace))).To(gomega.Succeed())
			g.Expect(artifactDeploymentList.Items).To(gomega.HaveLen(1))
			g.Expect(artifactDeploymentList.Items[0].Name).To(gomega.HavePrefix(artifactPrefix))
			artifactDeployment := &artifactDeploymentList.Items[0]
			g.Expect(artifactDeployment.Spec.Component.Resources).To(gomega.HaveLen(4))
		}, timeout, interval).Should(gomega.Succeed())

		By("Updating ArtifactDeployment.status")
		// Re-fetch to get the latest resourceVersion (the controller may have added an owner reference since the list).
		artifactDeployment := &landscape.ArtifactDeployment{}
		gomega.Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name:      artifactDeploymentList.Items[0].Name,
			Namespace: testNamespace,
		}, artifactDeployment)).To(gomega.Succeed())

		artifactDeployment.Status.DeploymentResults = []landscape.DeploymentResult{
			{Name: "result-1", Type: "test", Spec: runtime.RawExtension{Raw: []byte("{\"test-deployment-result\": true}")}},
		}
		meta.SetStatusCondition(&artifactDeployment.Status.Conditions, metav1.Condition{
			Type:               landscape.ArtifactDeployedCondition,
			Reason:             landscape.ArtifactDeployedCondition,
			Status:             metav1.ConditionTrue,
			Message:            "",
			ObservedGeneration: artifactDeployment.Generation,
			LastTransitionTime: metav1.Now(),
		})
		meta.SetStatusCondition(&artifactDeployment.Status.Conditions, metav1.Condition{
			Type:               landscape.DeploymentResultCreatedCondition,
			Reason:             landscape.DeploymentResultCreatedCondition,
			Status:             metav1.ConditionTrue,
			Message:            "",
			ObservedGeneration: artifactDeployment.Generation,
			LastTransitionTime: metav1.Now(),
		})
		meta.SetStatusCondition(&artifactDeployment.Status.Conditions, metav1.Condition{
			Type:               landscape.ArtifactDeploymentReadyCondition,
			Reason:             landscape.ArtifactDeploymentReadyCondition,
			Status:             metav1.ConditionTrue,
			Message:            "",
			ObservedGeneration: artifactDeployment.Generation,
			LastTransitionTime: metav1.Now(),
		})

		gomega.Expect(k8sClient.Status().Update(ctx, artifactDeployment)).To(gomega.Succeed())

		By("Verifying VectorDeployed condition is set")
		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: ocmName, Namespace: testNamespace}, actualVectorDeployment)).To(gomega.Succeed())
			g.Expect(meta.IsStatusConditionTrue(actualVectorDeployment.Status.Conditions, landscape.VectorDeployedCondition)).To(gomega.BeTrue())
		}, timeout, interval).Should(gomega.Succeed())

		By("Verifying VectorDeployment.status.deploymentResults is populated")
		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: ocmName, Namespace: testNamespace}, actualVectorDeployment)).To(gomega.Succeed())
			g.Expect(actualVectorDeployment.Status.DeploymentResults).To(gomega.HaveLen(1))
		}, timeout, interval).Should(gomega.Succeed())

		By("Verifying VectorAssignment was created")
		vectorAssignmentList := &landscape.VectorAssignmentList{}
		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(k8sClient.List(ctx, vectorAssignmentList, client.InNamespace(testNamespace))).To(gomega.Succeed())
			g.Expect(vectorAssignmentList.Items).To(gomega.HaveLen(1))

			vectorAssignment := vectorAssignmentList.Items[0]
			g.Expect(vectorAssignment.Name).To(gomega.Equal(vectorDeployment.Name))
			g.Expect(vectorAssignment.Spec.VectorDeploymentRef.Name).To(gomega.Equal(vectorDeployment.Name))
			g.Expect(vectorAssignment.Spec.ArtifactDeploymentRef.Name).To(gomega.Equal(artifactDeployment.Name))
		}, timeout, interval).Should(gomega.Succeed())

		By("Verifying VectorAssignmentsCreated condition is set")
		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: ocmName, Namespace: testNamespace}, actualVectorDeployment)).To(gomega.Succeed())
			g.Expect(meta.IsStatusConditionTrue(actualVectorDeployment.Status.Conditions, landscape.VectorAssignmentsCreatedCondition)).To(gomega.BeTrue())
		}, timeout, interval).Should(gomega.Succeed())
	})
})

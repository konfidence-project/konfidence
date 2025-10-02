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
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/konfidence-project/landscape-vector-deployment-controller/internal/controller/domain"
	"github.com/konfidence-project/landscape-vector-deployment-controller/internal/controller/domain/mocks"
)

var _ = Describe("VectorDeployment Controller", func() {

	const (
		controllerName  = "vectordeployment"
		vectorReference = "https://registry.kdenv.lab/sample-project//github.com/konfidence-project/sample-vector:0.3.0"
		ocmName         = "common.konfidence.cloud.example.vector-0.3.0"
		testNamespace   = "default"
		timeout         = time.Second * 10
		interval        = time.Millisecond * 250
	)

	var (
		reconciler     *VectorDeploymentReconciler
		ocmAdapterMock *mocks.MockVectorOcmPort
		mockCtrl       *gomock.Controller
	)

	BeforeEach(func() {
		// Mock setup
		mockCtrl = gomock.NewController(GinkgoT())
		ocmAdapterMock = mocks.NewMockVectorOcmPort(mockCtrl)

		// Controller setup
		reconciler = &VectorDeploymentReconciler{
			Client:     k8sManager.GetClient(),
			Scheme:     k8sManager.GetScheme(),
			OcmAdapter: ocmAdapterMock,
		}
		err := reconciler.SetupWithManager(k8sManager, controllerName)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	})

	AfterEach(func() {
		ctx := context.Background()

		managerClient := k8sManager.GetClient()

		// Cleanup VectorDeployments
		vectorDeploymentList := &landscape.VectorDeploymentList{}
		err := managerClient.List(ctx, vectorDeploymentList, client.InNamespace(testNamespace))
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		for _, vd := range vectorDeploymentList.Items {
			err := managerClient.Delete(ctx, &vd)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		}

		// Cleanup ArtifactDeployments
		artifactDeploymentList := &landscape.ArtifactDeploymentList{}
		err = managerClient.List(ctx, artifactDeploymentList, client.InNamespace(testNamespace))
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		for _, ad := range artifactDeploymentList.Items {
			err := managerClient.Delete(ctx, &ad)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
		}

		if mockCtrl != nil {
			mockCtrl.Finish()
		}
	})

	It("should successfully reconcile the vector-deployment resource", func() {
		ctx := context.Background()
		// when: GetVectorByReference is called on the ocmAdapter mock
		// then: mock should be called with a vector reference and return a Vector
		vector := domain.Vector{
			Reference: domain.VectorReference{
				OciRegistryUrl: "https://registry.kdenv.lab/sample-project//github.com/konfidence-project/sample-vector:0.3.0",
				Component:      "github.com/konfidence-project/sample-vector",
				Version:        "0.3.0",
			},
			ComponentSpec: "{\"name\":\"github.com/konfidence-project/sample-vector\",\"version\":\"0.3.0\",\"labels\":[{\"name\":\"konfidence-project/sample-vector\",\"value\":\"01904be8-bae3-ae70-e4d6-78af41d7e0a2\",\"version\":\"v1\"}],\"provider\":{\"name\":\"konfidence-project\"},\"creationTime\":\"2025-09-22T06:32:45Z\",\"repositoryContexts\":[{\"baseUrl\":\"https://registry.kdenv.lab\",\"componentNameMapping\":\"urlPath\",\"subPath\":\"sample-project\",\"type\":\"OCIRegistry\"}],\"sources\":[],\"componentReferences\":[{\"name\":\"sample-service-1\",\"version\":\"0.0.1\",\"componentName\":\"github.com/konfidence-project/sample-service-1\"}],\"resources\":[]}",
			Artifacts:     []domain.ArtifactReference{{Version: "0.0.1", ComponentName: "github.com/konfidence-project/sample-service-1"}},
		}
		ocmAdapterMock.EXPECT().GetVectorByReference(gomock.Any(), gomock.Any()).Return(&vector, nil).AnyTimes()

		// when: GetArtifactManifestByReference is called on the ocmAdapter mock
		// then: mock should be called with a vector reference and an artifact reference, and return a ArtifactManifest
		artifactManifest := domain.ArtifactManifest{
			Type:       "cloud.konfidence.flux.helm",
			AllowReuse: true,
			Tasks: []domain.TaskManifest{
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
			Resources: []domain.OCMResource{
				{
					Name: "sample-service-1-helm-chart",
					Type: "helmChart",
				},
				{
					Name: "sample-service-1-image",
					Type: "ociImage",
				},
				{
					Name: "konfidence-manifest",
					Type: "cloud.konfidence.artifact.manifest",
				},
				{
					Name: "sample-service-1-task-1-manifest",
					Type: "cloud.konfidence.artifact.task.manifest",
				},
				{
					Name: "sample-service-1-task-1",
					Type: "ociImage",
				},
				{
					Name: "sample-service-1-task-2-manifest",
					Type: "cloud.konfidence.artifact.task.manifest",
				},
				{
					Name: "sample-service-1-task-2",
					Type: "ociImage",
				},
			},
		}
		ocmAdapterMock.EXPECT().GetArtifactManifestByReference(gomock.Any(), vectorReference, vector.Artifacts[0]).Return(&artifactManifest, nil).AnyTimes()

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

		By("Verifying ResolvedVectorOcm and status conditions were set")
		actualVectorDeployment := &landscape.VectorDeployment{}
		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: ocmName, Namespace: testNamespace}, actualVectorDeployment)).To(gomega.Succeed())
			g.Expect(actualVectorDeployment.Status.ResolvedVectorOcm).To(gomega.Not(gomega.BeEmpty()))
			g.Expect(meta.IsStatusConditionTrue(actualVectorDeployment.Status.Conditions, landscape.VectorDownloadedCondition)).To(gomega.BeTrue())
			g.Expect(meta.IsStatusConditionTrue(actualVectorDeployment.Status.Conditions, landscape.ArtifactDeploymentsCreatedCondition)).To(gomega.BeTrue())
		}, timeout, interval).Should(gomega.Succeed())

		By("Verifying ArtifactDeployment was created")
		artifactDeploymentList := &landscape.ArtifactDeploymentList{}
		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(k8sClient.List(ctx, artifactDeploymentList, client.InNamespace(testNamespace))).To(gomega.Succeed())
			g.Expect(artifactDeploymentList.Items).To(gomega.HaveLen(1))

			artifactDeployment := artifactDeploymentList.Items[0]
			g.Expect(artifactDeployment.Spec.Component.Resources).To(gomega.HaveLen(7))
		}, timeout, interval).Should(gomega.Succeed())
	})

})

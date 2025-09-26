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
			Type:           "cloud.konfidence.flux.helm",
			AllowReuse:     true,
			OciRegistryUrl: "registry.kdenv.lab/sample-project//github.com/konfidence-project/sample-service-1:0.0.1",
			ComponentSpec:  "{\"name\":\"github.com/konfidence-project/sample-service-1\",\"version\":\"0.0.1\",\"provider\":{\"name\":\"konfidence-project\"},\"creationTime\":\"2025-09-22T06:32:37Z\",\"repositoryContexts\":[{\"baseUrl\":\"https://registry.kdenv.lab\",\"componentNameMapping\":\"urlPath\",\"subPath\":\"sample-project\",\"type\":\"OCIRegistry\"}],\"sources\":[],\"componentReferences\":[],\"resources\":[{\"name\":\"sample-service-1-helm-chart\",\"version\":\"6.9.1\",\"type\":\"helmChart\",\"relation\":\"external\",\"digest\":{\"hashAlgorithm\":\"SHA-256\",\"normalisationAlgorithm\":\"genericBlobDigest/v1\",\"value\":\"6d082dc0d4e90fbb525c0c1fc8a52d5279581750b8888688b07ce00f96d947e8\"},\"access\":{\"helmChart\":\"podinfo:6.9.1\",\"helmRepository\":\"https://stefanprodan.github.io/podinfo\",\"type\":\"helm\"}},{\"name\":\"sample-service-1-image\",\"version\":\"0.0.1\",\"type\":\"ociImage\",\"relation\":\"external\",\"digest\":{\"hashAlgorithm\":\"SHA-256\",\"normalisationAlgorithm\":\"ociArtifactDigest/v1\",\"value\":\"262578cde928d5c9eba3bce079976444f624c13ed0afb741d90d5423877496cb\"},\"access\":{\"imageReference\":\"stefanprodan/podinfo:6.9.1\",\"type\":\"ociArtifact\"}},{\"name\":\"konfidence-manifest\",\"version\":\"0.0.1\",\"type\":\"cloud.konfidence.artifact.manifest\",\"relation\":\"local\",\"digest\":{\"hashAlgorithm\":\"SHA-256\",\"normalisationAlgorithm\":\"genericBlobDigest/v1\",\"value\":\"7052d8b081b4158cb6832dc9cb91f0feb7faf23ee8d25704f8fef28ce6f3d7ea\"},\"access\":{\"localReference\":\"sha256:7052d8b081b4158cb6832dc9cb91f0feb7faf23ee8d25704f8fef28ce6f3d7ea\",\"mediaType\":\"application/octet-stream\",\"type\":\"localBlob\"}},{\"name\":\"sample-service-1-task-1-manifest\",\"version\":\"0.0.1\",\"type\":\"cloud.konfidence.artifact.task.manifest\",\"relation\":\"local\",\"digest\":{\"hashAlgorithm\":\"SHA-256\",\"normalisationAlgorithm\":\"genericBlobDigest/v1\",\"value\":\"f6b421d7e0bbc5f9a2b756f065e25413e6bfc9946099deee8b58f29cfa6ac4b6\"},\"access\":{\"localReference\":\"sha256:f6b421d7e0bbc5f9a2b756f065e25413e6bfc9946099deee8b58f29cfa6ac4b6\",\"mediaType\":\"application/octet-stream\",\"type\":\"localBlob\"}},{\"name\":\"sample-service-1-task-1\",\"version\":\"0.0.1\",\"type\":\"ociImage\",\"relation\":\"external\",\"digest\":{\"hashAlgorithm\":\"SHA-256\",\"normalisationAlgorithm\":\"ociArtifactDigest/v1\",\"value\":\"4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1\"},\"access\":{\"imageReference\":\"alpine:3.22.1\",\"type\":\"ociArtifact\"}},{\"name\":\"sample-service-1-task-2-manifest\",\"version\":\"0.0.1\",\"type\":\"cloud.konfidence.artifact.task.manifest\",\"relation\":\"local\",\"digest\":{\"hashAlgorithm\":\"SHA-256\",\"normalisationAlgorithm\":\"genericBlobDigest/v1\",\"value\":\"7ec3dbf60da751c33cb749d9ff7d615d3c58636b8e0516e1830ba802d83ce40b\"},\"access\":{\"localReference\":\"sha256:7ec3dbf60da751c33cb749d9ff7d615d3c58636b8e0516e1830ba802d83ce40b\",\"mediaType\":\"application/octet-stream\",\"type\":\"localBlob\"}},{\"name\":\"sample-service-1-task-2\",\"version\":\"0.0.1\",\"type\":\"cloud.konfidence.artifact.task\",\"relation\":\"external\",\"digest\":{\"hashAlgorithm\":\"SHA-256\",\"normalisationAlgorithm\":\"ociArtifactDigest/v1\",\"value\":\"4bcff63911fcb4448bd4fdacec207030997caf25e9bea4045fa6c8c44de311d1\"},\"access\":{\"imageReference\":\"alpine:3.22.1\",\"type\":\"ociArtifact\"}}]}",
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

		By("Verifying ResolvedVectorOcm was set")
		actualVectorDeployment := &landscape.VectorDeployment{}
		gomega.Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: ocmName, Namespace: testNamespace}, actualVectorDeployment)
			if err != nil {
				return false
			}
			return actualVectorDeployment.Status.ResolvedVectorOcm != ""
		}, timeout, interval).Should(gomega.BeTrue(), "VectorDeployment should be processed automatically")

		By("Verifying VectorDownloadedCondition was set to True")
		gomega.Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: ocmName, Namespace: testNamespace}, actualVectorDeployment)
			if err != nil {
				return false
			}
			for _, condition := range actualVectorDeployment.Status.Conditions {
				if condition.Type == landscape.VectorDownloadedCondition && condition.Status == metav1.ConditionTrue {
					return true
				}
			}
			return false
		}, timeout, interval).Should(gomega.BeTrue(), "VectorDownloadedCondition should be True")

		By("Verifying ArtifactDeploymentsCreatedCondition was set")
		gomega.Eventually(func() bool {
			err := k8sClient.Get(ctx, types.NamespacedName{Name: ocmName, Namespace: testNamespace}, actualVectorDeployment)
			if err != nil {
				return false
			}
			for _, condition := range actualVectorDeployment.Status.Conditions {
				if condition.Type == landscape.ArtifactDeploymentsCreatedCondition && condition.Status == metav1.ConditionTrue {
					return true
				}
			}
			return false
		}, timeout, interval).Should(gomega.BeTrue(), "ArtifactDeploymentsCreatedCondition should be True")

		By("Verifying ArtifactDeployment was created")
		gomega.Eventually(func() bool {
			artifactDeploymentList := &landscape.ArtifactDeploymentList{}
			err = k8sClient.List(ctx, artifactDeploymentList, client.InNamespace(testNamespace))
			return err == nil && len(artifactDeploymentList.Items) == 1
		}, timeout, interval).Should(gomega.BeTrue(), "Should have created exactly one ArtifactDeployment")
	})

})

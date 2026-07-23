package controller_test

import (
	"context"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/konfidence-project/konfidence/internal/vectordeployment/internal/controller"
	"github.com/konfidence-project/konfidence/internal/vectordeployment/internal/controller/mocks"
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

var _ = Describe("VectorDeployment Controller", Ordered, Serial, func() {

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
		ocmAdapterMock *mocks.MockVectorOcmPort
		mockCtrl       *gomock.Controller
	)

	BeforeAll(func() {
		reconciler = &VectorDeploymentReconciler{
			Client:   k8sManager.GetClient(),
			Scheme:   k8sManager.GetScheme(),
			Recorder: k8sManager.GetEventRecorder(VectorDeploymentControllerName),
		}
		gomega.Expect(reconciler.SetupWithManager(k8sManager, controllerName)).To(gomega.Succeed())
	})

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		ocmAdapterMock = mocks.NewMockVectorOcmPort(mockCtrl)
		reconciler.OcmAdapter = ocmAdapterMock
	})

	AfterEach(func() {
		ctx := context.Background()

		managerClient := k8sManager.GetClient()

		// Cleanup VectorDeployments
		err := managerClient.DeleteAllOf(ctx, &konfidence.VectorDeployment{}, client.InNamespace(testNamespace))
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		// Cleanup ArtifactDeployments
		err = managerClient.DeleteAllOf(ctx, &konfidence.ArtifactDeployment{}, client.InNamespace(testNamespace))
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		// Cleanup VectorAssignments
		err = managerClient.DeleteAllOf(ctx, &konfidence.VectorAssignment{}, client.InNamespace(testNamespace))
		gomega.Expect(err).ToNot(gomega.HaveOccurred())

		// Cleanup VectorData CRs. The owner-reference cascade from the VD deletion above would handle this
		// eventually, but a manual sweep keeps the suite deterministic for re-runs.
		err = managerClient.DeleteAllOf(ctx, &konfidence.VectorData{}, client.InNamespace(testNamespace))
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
			DescriptorJSON: []byte(`{"meta":{"schemaVersion":"v2"},"component":{"name":"github.com/konfidence-project/sample-vector",` +
				`"version":"0.3.0","labels":[{"name":"konfidence-project/sample-vector",` +
				`"value":"01904be8-bae3-ae70-e4d6-78af41d7e0a2","version":"v1"}],` +
				`"creationTime":"2025-09-22T06:32:45Z","repositoryContexts":null,"provider":"konfidence-project",` +
				`"resources":[],"sources":[],"componentReferences":[{"name":"sample-service-1","version":"0.0.1",` +
				`"componentName":"github.com/konfidence-project/sample-service-1",` +
				`"digest":{"hashAlgorithm":"","normalisationAlgorithm":"","value":""}}]}}`),
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
					Spec: "{\"template\":{\"spec\":{\"restartPolicy\":\"Never\"," +
						"\"containers\":[{\"name\":\"sample-service-1-task-1-container\"," +
						"\"image\":\"alpine:3.22.1\",\"command\":[\"echo\"," +
						"\"I am task 1 of service 1\"]}]}},\"backoffLimit\":4}",
				},
				{
					Name:      "sample-service-1-task-2",
					Type:      "k8s-job",
					DependsOn: nil,
					Spec: "{\"template\":{\"spec\":{\"restartPolicy\":\"Never\"," +
						"\"containers\":[{\"name\":\"sample-service-1-task-2-container\"," +
						"\"image\":\"alpine:3.22.1\",\"command\":[\"echo\"," +
						"\"I am task 2 of service 1\"]}]}},\"backoffLimit\":4}",
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
		vectorDeployment := &konfidence.VectorDeployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       "VectorDeployment",
				APIVersion: "konfidence.cloud/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      ocmName,
				Namespace: testNamespace,
				Labels:    map[string]string{"app.kubernetes.io/name": "crds"},
			},
			Spec: konfidence.VectorDeploymentSpec{
				Vector: vectorReference,
			},
			Status: konfidence.VectorDeploymentStatus{},
		}

		// WHEN: creating the resource will trigger the reconciler automatically
		err := k8sClient.Create(ctx, vectorDeployment)
		gomega.Expect(err).To(gomega.Succeed())
		By("successfully created VectorDeployment resource")

		// THEN: Verify that the reconciler processed the resource and updated

		By("Verifying ResolvedVectorOcm and VectorDownloaded condition is set")
		actualVectorDeployment := &konfidence.VectorDeployment{}
		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: ocmName, Namespace: testNamespace}, actualVectorDeployment)).To(gomega.Succeed())
			g.Expect(actualVectorDeployment.Status.ResolvedVectorOcm).To(gomega.Not(gomega.BeEmpty()))
			g.Expect(meta.IsStatusConditionTrue(actualVectorDeployment.Status.Conditions, konfidence.VectorDownloadedCondition)).To(gomega.BeTrue())
		}, timeout, interval).Should(gomega.Succeed())

		By("Verifying ArtifactDeployment was created")
		artifactDeploymentList := &konfidence.ArtifactDeploymentList{}
		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(k8sClient.List(ctx, artifactDeploymentList, client.InNamespace(testNamespace))).To(gomega.Succeed())
			g.Expect(artifactDeploymentList.Items).To(gomega.HaveLen(1))
			g.Expect(artifactDeploymentList.Items[0].Name).To(gomega.HavePrefix(artifactPrefix))
			artifactDeployment := &artifactDeploymentList.Items[0]
			g.Expect(artifactDeployment.Spec.Component.Resources).To(gomega.HaveLen(4))
		}, timeout, interval).Should(gomega.Succeed())

		By("Updating ArtifactDeployment.status")
		// Re-fetch to get the latest resourceVersion (the controller may have added an owner reference since the list).
		artifactDeployment := &konfidence.ArtifactDeployment{}
		gomega.Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name:      artifactDeploymentList.Items[0].Name,
			Namespace: testNamespace,
		}, artifactDeployment)).To(gomega.Succeed())

		artifactDeployment.Status.DeploymentResults = []konfidence.DeploymentResult{
			{Name: "result-1", Type: "test", Spec: runtime.RawExtension{Raw: []byte("{\"test-deployment-result\": true}")}},
		}
		meta.SetStatusCondition(&artifactDeployment.Status.Conditions, metav1.Condition{
			Type:               konfidence.ArtifactDeployedCondition,
			Reason:             konfidence.ArtifactDeployedCondition,
			Status:             metav1.ConditionTrue,
			Message:            "",
			ObservedGeneration: artifactDeployment.Generation,
			LastTransitionTime: metav1.Now(),
		})
		meta.SetStatusCondition(&artifactDeployment.Status.Conditions, metav1.Condition{
			Type:               konfidence.DeploymentResultCreatedCondition,
			Reason:             konfidence.DeploymentResultCreatedCondition,
			Status:             metav1.ConditionTrue,
			Message:            "",
			ObservedGeneration: artifactDeployment.Generation,
			LastTransitionTime: metav1.Now(),
		})
		meta.SetStatusCondition(&artifactDeployment.Status.Conditions, metav1.Condition{
			Type:               konfidence.ArtifactDeploymentReadyCondition,
			Reason:             konfidence.ArtifactDeploymentReadyCondition,
			Status:             metav1.ConditionTrue,
			Message:            "",
			ObservedGeneration: artifactDeployment.Generation,
			LastTransitionTime: metav1.Now(),
		})

		gomega.Expect(k8sClient.Status().Update(ctx, artifactDeployment)).To(gomega.Succeed())

		By("Verifying VectorDeployed condition is set")
		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: ocmName, Namespace: testNamespace}, actualVectorDeployment)).To(gomega.Succeed())
			g.Expect(meta.IsStatusConditionTrue(actualVectorDeployment.Status.Conditions, konfidence.VectorDeployedCondition)).To(gomega.BeTrue())
		}, timeout, interval).Should(gomega.Succeed())

		By("Verifying VectorDeployment.status.deploymentResults is populated")
		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: ocmName, Namespace: testNamespace}, actualVectorDeployment)).To(gomega.Succeed())
			g.Expect(actualVectorDeployment.Status.DeploymentResults).To(gomega.HaveLen(1))
		}, timeout, interval).Should(gomega.Succeed())

		By("Verifying VectorAssignment was created")
		vectorAssignmentList := &konfidence.VectorAssignmentList{}
		vectorAssignmentName := ""
		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(k8sClient.List(ctx, vectorAssignmentList, client.InNamespace(testNamespace))).To(gomega.Succeed())
			g.Expect(vectorAssignmentList.Items).To(gomega.HaveLen(1))

			vectorAssignment := vectorAssignmentList.Items[0]
			vectorAssignmentName = vectorAssignment.Name
			g.Expect(vectorAssignment.Spec.VectorDeploymentRef.Name).To(gomega.Equal(vectorDeployment.Name))
			g.Expect(vectorAssignment.Spec.ArtifactDeploymentRef.Name).To(gomega.Equal(artifactDeployment.Name))
		}, timeout, interval).Should(gomega.Succeed())

		By("Verifying VectorAssignmentsCreated condition is set")
		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: ocmName, Namespace: testNamespace}, actualVectorDeployment)).To(gomega.Succeed())
			g.Expect(meta.IsStatusConditionTrue(actualVectorDeployment.Status.Conditions, konfidence.VectorAssignmentsCreatedCondition)).To(gomega.BeTrue())
		}, timeout, interval).Should(gomega.Succeed())

		By("Verifying ArtifactDeployment has owner reference pointing to VectorDeployment")
		gomega.Eventually(func(g gomega.Gomega) {
			ad := &konfidence.ArtifactDeployment{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      artifactDeploymentList.Items[0].Name,
				Namespace: testNamespace,
			}, ad)).To(gomega.Succeed())
			g.Expect(ad.OwnerReferences).ToNot(gomega.BeEmpty())
			foundOwner := false
			for _, ref := range ad.OwnerReferences {
				if ref.UID == vectorDeployment.UID && ref.Kind == konfidence.VectorDeploymentKind {
					foundOwner = true
				}
			}
			g.Expect(foundOwner).To(gomega.BeTrue(), "ArtifactDeployment should have VectorDeployment as owner")
		}, timeout, interval).Should(gomega.Succeed())

		By("Verifying VectorAssignment has controller owner reference pointing to VectorDeployment")
		gomega.Eventually(func(g gomega.Gomega) {
			va := &konfidence.VectorAssignment{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      vectorAssignmentName,
				Namespace: testNamespace,
			}, va)).To(gomega.Succeed())
			g.Expect(va.OwnerReferences).ToNot(gomega.BeEmpty())
			foundControllerOwner := false
			for _, ref := range va.OwnerReferences {
				if ref.UID == vectorDeployment.UID && ref.Kind == konfidence.VectorDeploymentKind && ref.Controller != nil && *ref.Controller {
					foundControllerOwner = true
				}
			}
			g.Expect(foundControllerOwner).To(gomega.BeTrue(), "VectorAssignment should have VectorDeployment as controller owner")
		}, timeout, interval).Should(gomega.Succeed())

		By("Simulating VectorAssignment readiness")
		vectorAssignment := &konfidence.VectorAssignment{}
		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      vectorAssignmentName,
				Namespace: testNamespace,
			}, vectorAssignment)).To(gomega.Succeed())
		}, timeout, interval).Should(gomega.Succeed())

		meta.SetStatusCondition(&vectorAssignment.Status.Conditions, metav1.Condition{
			Type:               konfidence.VectorAssignmentReadyCondition,
			Reason:             konfidence.VectorAssignmentReadyCondition,
			Status:             metav1.ConditionTrue,
			Message:            "simulated",
			ObservedGeneration: vectorAssignment.Generation,
			LastTransitionTime: metav1.Now(),
		})
		gomega.Expect(k8sClient.Status().Update(ctx, vectorAssignment)).To(gomega.Succeed())

		By("Verifying VectorData CR was created with the inlined payload")
		// The vector deployment controller no longer materialises the ConfigMap directly. Its responsibility now ends at
		// emitting a runtime-agnostic VectorData CR with the OCM-resolved bytes + aggregated DeploymentResults.
		// A runtime-specific orchestrator (e.g. kubernetes-landscape-orchestrator) consumes the CR and writes the
		// ConfigMap (or another runtime-shaped artefact). We simulate the orchestrator below.
		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: ocmName, Namespace: testNamespace}, actualVectorDeployment)).To(gomega.Succeed())
			g.Expect(meta.IsStatusConditionTrue(actualVectorDeployment.Status.Conditions, konfidence.VectorDataCreatedCondition)).To(gomega.BeTrue())
			g.Expect(actualVectorDeployment.Status.ResultingVectorData).ToNot(gomega.BeNil())
			g.Expect(actualVectorDeployment.Status.ResultingVectorData.Name).To(gomega.Equal(ocmName))

			vectorData := &konfidence.VectorData{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: ocmName, Namespace: testNamespace}, vectorData)).To(gomega.Succeed())

			By("VectorData inlines the aggregated DeploymentResults from all ArtifactDeployments")
			g.Expect(vectorData.Spec.DeploymentResults).To(gomega.HaveLen(1))

			By("VectorData has Spec.Features/Spec.Authored nil because this vector did not declare a vector-config OCM resource")
			g.Expect(vectorData.Spec.Features).To(gomega.BeNil())
			g.Expect(vectorData.Spec.Authored).To(gomega.BeNil())

			By("VectorData is owned by the VectorDeployment for cascade-delete")
			g.Expect(vectorData.OwnerReferences).ToNot(gomega.BeEmpty())
			foundOwner := false
			for _, ref := range vectorData.OwnerReferences {
				if ref.UID == actualVectorDeployment.UID && ref.Kind == konfidence.VectorDeploymentKind && ref.Controller != nil && *ref.Controller {
					foundOwner = true
				}
			}
			g.Expect(foundOwner).To(gomega.BeTrue(), "VectorData should be controller-owned by VectorDeployment")
		}, timeout, interval).Should(gomega.Succeed())

		By("Simulating the landscape orchestrator: flipping VectorData.Status.Ready=True")
		// Without the orchestrator the VectorDeployment lifecycle stalls at "waiting for VectorData to be
		// materialized" — which is itself the correct, desired behaviour. We unblock it by hand here so the rest
		// of the assertions run.
		gomega.Eventually(func(g gomega.Gomega) {
			vectorData := &konfidence.VectorData{}
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: ocmName, Namespace: testNamespace}, vectorData)).To(gomega.Succeed())
			meta.SetStatusCondition(&vectorData.Status.Conditions, metav1.Condition{
				Type:               konfidence.VectorDataReadyCondition,
				Status:             metav1.ConditionTrue,
				Reason:             konfidence.VectorDataReasonMaterialized,
				Message:            "simulated by envtest",
				ObservedGeneration: vectorData.Generation,
				LastTransitionTime: metav1.Now(),
			})
			g.Expect(k8sClient.Status().Update(ctx, vectorData)).To(gomega.Succeed())
		}, timeout, interval).Should(gomega.Succeed())

		By("Verifying VectorReady condition is set on VectorDeployment")
		gomega.Eventually(func(g gomega.Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: ocmName, Namespace: testNamespace}, actualVectorDeployment)).To(gomega.Succeed())
			g.Expect(meta.IsStatusConditionTrue(actualVectorDeployment.Status.Conditions, konfidence.VectorReadyCondition)).To(gomega.BeTrue())
		}, timeout, interval).Should(gomega.Succeed())
	})
})

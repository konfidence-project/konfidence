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

	"github.com/aws/smithy-go/ptr"
	global "github.com/konfidence-project/crds/api/global/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/konfidence-project/gcp-vector-assembly-controller/internal/controller/domain"
	"github.com/konfidence-project/gcp-vector-assembly-controller/internal/controller/domain/mocks"
)

var _ = Describe("Vector Assembly Controller", Ordered, func() {

	const (
		testNamespace = "default"
		timeout       = time.Second * 10
		interval      = time.Millisecond * 250
	)

	var (
		ocmAdapterMock *mocks.MockVectorOcmPort
		mockCtrl       *gomock.Controller
		k8sClient      client.Client
		cancel         context.CancelFunc
	)

	BeforeAll(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		ocmAdapterMock = mocks.NewMockVectorOcmPort(mockCtrl)

		k8sClient, cancel = StartTestManagerWithReconciler(func(mgr ctrl.Manager) error {
			return (&VectorTemplateReconciler{
				Client:     mgr.GetClient(),
				Scheme:     mgr.GetScheme(),
				OcmAdapter: ocmAdapterMock,
				Recorder:   mgr.GetEventRecorder(VectorAssemblyControllerName),
			}).SetupWithManager(mgr)
		})
	})

	AfterAll(func() {
		if mockCtrl != nil {
			mockCtrl.Finish()
		}
		cancel()
	})

	BeforeEach(func() {
		By("clearing existing VectorTemplate CRs before each test")
		ctx := context.Background()
		err := k8sClient.DeleteAllOf(ctx, &global.VectorTemplate{}, client.InNamespace(testNamespace))
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	})

	It("should successfully reconcile a VectorTemplate CR for an existing vector", func() {
		ctx := context.Background()

		// when: GetLatestArtifactVersions is called
		// then: return latestComponentVersions stored in registry
		ocmComponentsFromComponentList := []domain.OcmReference{
			{
				Component:  "dwc.tools.sap/dwc-project/dev/service1",
				Repository: "localhost:5100",
			},
			{
				Component:  "dwc.tools.sap/dwc-project/dev/service2",
				Repository: "localhost:5100",
			},
		}
		latestComponentVersions := []domain.Artifact{
			{
				Version: "1.2.0",
				OcmReference: domain.OcmReference{
					Component:  "dwc.tools.sap/dwc-project/dev/service1",
					Repository: "localhost:5100",
				},
			},
			{
				Version: "3.1.0", // this indicates a new version is available for service2
				OcmReference: domain.OcmReference{
					Component:  "dwc.tools.sap/dwc-project/dev/service2",
					Repository: "localhost:5100",
				},
			},
		}
		ocmAdapterMock.EXPECT().GetLatestArtifactVersions(gomock.Any(), ocmComponentsFromComponentList).Return(latestComponentVersions, nil).Times(1)

		// when: GetLatestVector is called
		// then: return an latest vector stored in registry
		latestVector := domain.Vector{
			Version: "2026.1.30-083712000Z",
			Reference: domain.OcmReference{
				Component:  "konfidence.cloud/sample-vector/sample-app",
				Repository: "localhost:5100",
			},
			Artifacts: []domain.Artifact{
				{
					Version: "1.2.0",
					OcmReference: domain.OcmReference{
						Component:  "dwc.tools.sap/dwc-project/dev/service1",
						Repository: "localhost:5100",
					},
				},
				{
					Version: "2.99.0", // older version for service2
					OcmReference: domain.OcmReference{
						Component:  "dwc.tools.sap/dwc-project/dev/service2",
						Repository: "localhost:5100",
					},
				},
			},
		}
		ocmAdapterMock.EXPECT().GetLatestVector(gomock.Any(), gomock.Any()).Return(latestVector, nil).Times(1)

		// when: CreateVector is called
		// then: expect it to be called with updated vector containing new version for service2
		expectedUpdatedVector := domain.Vector{
			Version: "NOT-TESTABLE", // version can't be verified, because it's generated based on current timestamp
			Reference: domain.OcmReference{
				Component:  "konfidence.cloud/sample-vector/sample-app",
				Repository: "localhost:5100",
			},
			Artifacts: []domain.Artifact{
				{
					Version: "1.2.0",
					OcmReference: domain.OcmReference{
						Component:  "dwc.tools.sap/dwc-project/dev/service1",
						Repository: "localhost:5100",
					},
				},
				{
					Version: "3.1.0", // updated version for service2
					OcmReference: domain.OcmReference{
						Component:  "dwc.tools.sap/dwc-project/dev/service2",
						Repository: "localhost:5100",
					},
				},
			},
		}
		ocmAdapterMock.EXPECT().CreateVector(gomock.Any(), gomock.AssignableToTypeOf(domain.Vector{})).
			DoAndReturn(func(ctx context.Context, v domain.Vector) error {
				gomega.Expect(v.Reference).To(gomega.Equal(expectedUpdatedVector.Reference))
				gomega.Expect(v.Artifacts).To(gomega.Equal(expectedUpdatedVector.Artifacts))
				return nil
			}).
			Times(1)

		// GIVEN: a VectorTemplate CR
		vectorTemplate := getDefaultVectorTemplateCR(testNamespace)

		// WHEN: creating a VectorTemplate CR will trigger the reconciler automatically
		err := k8sClient.Create(ctx, vectorTemplate)
		gomega.Expect(err).To(gomega.Succeed())
		By("successfully created VectorTemplate resource")

		// THEN: Verify that the reconciler processed the vectorTemplate successfully
		By("Verify that the VectorTemplateStatus is updated")
		gomega.Eventually(func(g gomega.Gomega) {
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(vectorTemplate), vectorTemplate)
			g.Expect(err).NotTo(gomega.HaveOccurred())

			// verify ready condition
			readyCondition := meta.FindStatusCondition(vectorTemplate.Status.Conditions, global.VectorTemplateReadyCondition)
			g.Expect(readyCondition).NotTo(gomega.BeNil(), "Ready condition should be set")
			g.Expect(readyCondition.Status).To(gomega.Equal(metav1.ConditionTrue))
			g.Expect(readyCondition.Reason).To(gomega.Equal(global.VectorTemplateVectorCreatedReason))
			g.Expect(readyCondition.ObservedGeneration).To(gomega.Equal(vectorTemplate.Generation))
		}, timeout, interval).Should(gomega.Succeed())
	})

	It("should reconcile a VectorTemplate CR with a inheritance vector even if the vector does not exist in registry", func() {
		ctx := context.Background()

		// when: GetLatestVector is called for the base vector
		// then: return an latest base vector stored in registry
		baseVectorOcmReference := domain.OcmReference{
			Component:  "konfidence.project/common-services",
			Repository: "localhost:5100",
		}
		baseVector := domain.Vector{
			Version:   "2026.1.15-101112000Z",
			Reference: baseVectorOcmReference,
			Artifacts: []domain.Artifact{
				{
					Version: "0.9.0",
					OcmReference: domain.OcmReference{
						Component:  "dwc.tools.sap/dwc-project/dev/service3", // service from base vector
						Repository: "localhost:5100",
					},
				},
			},
		}
		ocmAdapterMock.EXPECT().GetLatestVector(gomock.Any(), baseVectorOcmReference).Return(baseVector, nil).Times(1)

		// when: GetLatestArtifactVersions is called
		// then: return predefined latest versions for components
		ocmComponentsFromComponentList := []domain.OcmReference{
			{
				Component:  "dwc.tools.sap/dwc-project/dev/service1",
				Repository: "localhost:5100",
			},
			{
				Component:  "dwc.tools.sap/dwc-project/dev/service2",
				Repository: "localhost:5100",
			},
		}
		latestComponentVersions := []domain.Artifact{
			{
				Version: "1.2.0",
				OcmReference: domain.OcmReference{
					Component:  "dwc.tools.sap/dwc-project/dev/service1",
					Repository: "localhost:5100",
				},
			},
			{
				Version: "3.1.0",
				OcmReference: domain.OcmReference{
					Component:  "dwc.tools.sap/dwc-project/dev/service2",
					Repository: "localhost:5100",
				},
			},
		}
		ocmAdapterMock.EXPECT().GetLatestArtifactVersions(gomock.Any(), ocmComponentsFromComponentList).Return(latestComponentVersions, nil).Times(1)

		// when: GetLatestVector is called
		// then: return an error indicating that the vector does not exist
		ocmAdapterMock.EXPECT().GetLatestVector(gomock.Any(), gomock.Any()).
			Return(domain.Vector{}, domain.ErrVectorNotFound).Times(1)

		// when: CreateVector is called
		// then: expect it to be called to create the new vector
		expectedUpdatedVector := domain.Vector{
			Version: "NOT-TESTABLE", // version can't be verified, because it's generated based on current timestamp
			Reference: domain.OcmReference{
				Component:  "konfidence.cloud/sample-vector/sample-app",
				Repository: "localhost:5100",
			},
			Artifacts: []domain.Artifact{
				{
					Version: "0.9.0",
					OcmReference: domain.OcmReference{
						Component:  "dwc.tools.sap/dwc-project/dev/service3", // service 3 from base vector
						Repository: "localhost:5100",
					},
				},
				{
					Version: "1.2.0",
					OcmReference: domain.OcmReference{
						Component:  "dwc.tools.sap/dwc-project/dev/service1",
						Repository: "localhost:5100",
					},
				},
				{
					Version: "3.1.0", // updated version for service2
					OcmReference: domain.OcmReference{
						Component:  "dwc.tools.sap/dwc-project/dev/service2",
						Repository: "localhost:5100",
					},
				},
			},
		}
		ocmAdapterMock.EXPECT().CreateVector(gomock.Any(), gomock.AssignableToTypeOf(domain.Vector{})).
			DoAndReturn(func(ctx context.Context, v domain.Vector) error {
				gomega.Expect(v.Reference).To(gomega.Equal(expectedUpdatedVector.Reference))
				gomega.Expect(v.Artifacts).To(gomega.Equal(expectedUpdatedVector.Artifacts))
				return nil
			}).
			Times(1)

		// GIVEN: a VectorTemplate CR
		vectorTemplate := getDefaultVectorTemplateCR(testNamespace)
		vectorTemplate.Spec.Base = ptr.String("http://localhost:5100//konfidence.project/common-services")

		// WHEN: creating a VectorTemplate CR will trigger the reconciler automatically
		err := k8sClient.Create(ctx, vectorTemplate)
		gomega.Expect(err).To(gomega.Succeed())
		By("successfully created VectorTemplate resource")

		// THEN: Verify that the reconciler processed the vectorTemplate successfully
		By("Verify that the VectorTemplateStatus is updated")
		gomega.Eventually(func(g gomega.Gomega) {
			err := k8sClient.Get(ctx, client.ObjectKeyFromObject(vectorTemplate), vectorTemplate)
			g.Expect(err).NotTo(gomega.HaveOccurred())

			// verify ready condition
			readyCondition := meta.FindStatusCondition(vectorTemplate.Status.Conditions, global.VectorTemplateReadyCondition)
			g.Expect(readyCondition).NotTo(gomega.BeNil(), "Ready condition should be set")
			g.Expect(readyCondition.Status).To(gomega.Equal(metav1.ConditionTrue))
			g.Expect(readyCondition.Reason).To(gomega.Equal(global.VectorTemplateVectorCreatedReason))
			g.Expect(readyCondition.ObservedGeneration).To(gomega.Equal(vectorTemplate.Generation))
		}, timeout, interval).Should(gomega.Succeed())
	})
})

func getDefaultVectorTemplateCR(namespace string) *global.VectorTemplate {
	return &global.VectorTemplate{
		TypeMeta: metav1.TypeMeta{
			Kind:       "VectorTemplate",
			APIVersion: "global.konfidence.cloud/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-vector",
			Namespace: namespace,
		},
		Spec: global.VectorTemplateSpec{
			ReconcileInterval: &metav1.Duration{
				Duration: time.Second * 30,
			},
			UploadTarget: "http://localhost:5100//konfidence.cloud/sample-vector/sample-app",
			Base:         nil,
			Components: []global.Component{
				{
					Name: "http://localhost:5100//dwc.tools.sap/dwc-project/dev/service1",
				},
				{
					Name: "http://localhost:5100//dwc.tools.sap/dwc-project/dev/service2",
				},
			},
		},
		Status: global.VectorTemplateStatus{},
	}
}

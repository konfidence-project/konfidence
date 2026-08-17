package controller_test

import (
	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// The ArtifactDeployment CRD marks Status.DeploymentResults as a list-map keyed by (name, type). These specs prove
// the apiserver enforces that key, independent of any controller logic. A dedicated namespace keeps the created
// ArtifactDeployments out of the shared default namespace other specs assert on.
var _ = Describe("ArtifactDeployment deployment-result uniqueness", func() {
	const drNamespace = "dr-uniqueness"

	BeforeEach(func() {
		err := k8sClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: drNamespace}})
		if err != nil && !apierrors.IsAlreadyExists(err) {
			Expect(err).NotTo(HaveOccurred())
		}
	})

	newAD := func(name string) *konfidence.ArtifactDeployment {
		return &konfidence.ArtifactDeployment{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: drNamespace},
			Spec: konfidence.ArtifactDeploymentSpec{
				Manifest:      konfidence.ArtifactManifest{Type: "cloud.konfidence.flux.helm"},
				TaskManifests: []konfidence.TaskManifest{},
				Component:     konfidence.OCMComponent{Name: "github.com/acme/svc", Version: "1.0.0"},
			},
		}
	}
	result := func(name, typ, k8sName string) konfidence.DeploymentResult {
		return konfidence.DeploymentResult{
			Name: name,
			Type: typ,
			Spec: runtime.RawExtension{Raw: []byte(`{"K8sName":"` + k8sName + `"}`)},
		}
	}

	It("rejects two results with the same (name, type)", func() {
		ad := newAD("ad-dup-nametype")
		Expect(k8sClient.Create(ctx, ad)).To(Succeed())
		ad.Status.DeploymentResults = []konfidence.DeploymentResult{
			result("candidates", "http-k8s-service", "candidates-a"),
			result("candidates", "http-k8s-service", "candidates-b"),
		}
		Expect(k8sClient.Status().Update(ctx, ad)).ToNot(Succeed())
	})

	It("allows the same name under different types", func() {
		ad := newAD("ad-same-name-diff-type")
		Expect(k8sClient.Create(ctx, ad)).To(Succeed())
		ad.Status.DeploymentResults = []konfidence.DeploymentResult{
			result("orders", "http-k8s-service", "orders-a"),
			result("orders", "grpc-k8s-service", "orders-b"),
		}
		Expect(k8sClient.Status().Update(ctx, ad)).To(Succeed())
	})

	// VectorData/VectorDeployment carry map[string][]DeploymentResult, which cannot use a list-map key. A CEL rule
	// enforces (name, type) uniqueness within each component instead, guarding against writers that bypass the
	// aggregation path.
	newVectorData := func(name string, results map[string]konfidence.ComponentDeploymentResults) *konfidence.VectorData {
		return &konfidence.VectorData{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: drNamespace},
			Spec:       konfidence.VectorDataSpec{DeploymentResults: results},
		}
	}

	It("rejects a VectorData whose component results collide on (name, type)", func() {
		vd := newVectorData("vd-dup", map[string]konfidence.ComponentDeploymentResults{
			"github.com/acme/svc": {
				result("candidates", "http-k8s-service", "candidates-a"),
				result("candidates", "http-k8s-service", "candidates-b"),
			},
		})
		Expect(k8sClient.Create(ctx, vd)).ToNot(Succeed())
	})

	It("allows a VectorData with unique (name, type) per component", func() {
		vd := newVectorData("vd-ok", map[string]konfidence.ComponentDeploymentResults{
			"github.com/acme/svc": {
				result("orders", "http-k8s-service", "orders-a"),
				result("orders", "grpc-k8s-service", "orders-b"),
			},
		})
		Expect(k8sClient.Create(ctx, vd)).To(Succeed())
	})
})

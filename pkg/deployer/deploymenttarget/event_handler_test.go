package deploymenttarget

import (
	"context"

	konfidencev1alpha1 "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("requestsForReference", func() {
	It("returns only targets referencing the matching object", func() {
		scheme := runtime.NewScheme()
		Expect(konfidencev1alpha1.AddToScheme(scheme)).To(Succeed())
		gvk := corev1.SchemeGroupVersion.WithKind("Secret")
		targets := []client.Object{
			referenceTarget("matching", "landscape", "helm.example.com", "Secret", "credentials"),
			referenceTarget("other-name", "landscape", "helm.example.com", "Secret", "another"),
			referenceTarget("other-kind", "landscape", "helm.example.com", "ConfigMap", "credentials"),
			referenceTarget("unsupported", "landscape", "kustomize.example.com", "Secret", "credentials"),
			referenceTarget("other-namespace", "other", "helm.example.com", "Secret", "credentials"),
		}
		objects := make([]runtime.Object, len(targets))
		for i := range targets {
			objects[i] = targets[i]
		}
		c := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(objects...).Build()
		secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "credentials", Namespace: "landscape"}}
		requests, err := requestsForReference(
			context.Background(), c, secret, gvk,
			konfidencev1alpha1.ArtifactTypeSet{"helm.example.com": {}},
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(requests).To(ConsistOf(reconcile.Request{NamespacedName: client.ObjectKey{Namespace: "landscape", Name: "matching"}}))
	})
})

func referenceTarget(name, namespace string, artifactType konfidencev1alpha1.ArtifactType, kind, refName string) *konfidencev1alpha1.DeploymentTarget {
	return &konfidencev1alpha1.DeploymentTarget{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: konfidencev1alpha1.DeploymentTargetSpec{
			ArtifactType: artifactType,
			Parameters:   &konfidencev1alpha1.Parameters{Kind: kind, Name: refName},
		},
	}
}

var _ = Describe("EnqueueArtifactDeploymentsForTarget", func() {
	It("enqueues matching artifact deployments in the changed target's namespace", func() {
		const artifactType konfidencev1alpha1.ArtifactType = "helm.example.com"
		scheme := runtime.NewScheme()
		Expect(konfidencev1alpha1.AddToScheme(scheme)).To(Succeed())
		objects := []runtime.Object{
			artifactDeployment("matching-one", "landscape", artifactType),
			artifactDeployment("matching-two", "landscape", artifactType),
			artifactDeployment("other-type", "landscape", "kustomize.example.com"),
			artifactDeployment("other-namespace", "elsewhere", artifactType),
		}
		c := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(objects...).Build()
		handler := EnqueueArtifactDeploymentsForTarget(artifactType, c)
		requests := mapTargetEvent(handler, &konfidencev1alpha1.DeploymentTarget{
			ObjectMeta: metav1.ObjectMeta{Name: "target", Namespace: "landscape"},
			Spec:       konfidencev1alpha1.DeploymentTargetSpec{ArtifactType: artifactType},
		})

		Expect(requests).To(ConsistOf(
			reconcile.Request{NamespacedName: client.ObjectKey{Namespace: "landscape", Name: "matching-one"}},
			reconcile.Request{NamespacedName: client.ObjectKey{Namespace: "landscape", Name: "matching-two"}},
		))
	})

	It("does not list deployments for non-targets or targets with another artifact type", func() {
		const artifactType konfidencev1alpha1.ArtifactType = "helm.example.com"
		scheme := runtime.NewScheme()
		Expect(konfidencev1alpha1.AddToScheme(scheme)).To(Succeed())
		c := fake.NewClientBuilder().WithScheme(scheme).Build()
		handler := EnqueueArtifactDeploymentsForTarget(artifactType, c)

		Expect(mapTargetEvent(handler, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "credentials", Namespace: "landscape"},
		})).To(BeEmpty())
		Expect(mapTargetEvent(handler, &konfidencev1alpha1.DeploymentTarget{
			ObjectMeta: metav1.ObjectMeta{Name: "target", Namespace: "landscape"},
			Spec:       konfidencev1alpha1.DeploymentTargetSpec{ArtifactType: "kustomize.example.com"},
		})).To(BeEmpty())
	})
})

func artifactDeployment(name, namespace string, artifactType konfidencev1alpha1.ArtifactType) *konfidencev1alpha1.ArtifactDeployment {
	return &konfidencev1alpha1.ArtifactDeployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec:       konfidencev1alpha1.ArtifactDeploymentSpec{ArtifactType: artifactType},
	}
}

func mapTargetEvent(h handler.EventHandler, obj client.Object) []reconcile.Request {
	q := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[reconcile.Request]())
	h.Generic(context.Background(), event.TypedGenericEvent[client.Object]{Object: obj}, q)
	var requests []reconcile.Request
	for q.Len() > 0 {
		request, shutdown := q.Get()
		if shutdown {
			break
		}
		requests = append(requests, request)
		q.Done(request)
	}
	return requests
}

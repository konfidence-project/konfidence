package controller

import (
	"context"
	"fmt"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("DeploymentTarget reconciliation", func() {
	ctx := context.Background()
	newTarget := func(name, class string, ready *metav1.Condition) *konfidence.DeploymentTarget {
		target := &konfidence.DeploymentTarget{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "landscape"},
			Spec:       konfidence.DeploymentTargetSpec{DeploymentClassName: class, Connection: konfidence.DeploymentTargetConnection{Type: "test"}},
		}
		if ready != nil {
			target.Status.Conditions = []metav1.Condition{*ready}
		}
		return target
	}
	ready := func() *metav1.Condition {
		return &metav1.Condition{Type: konfidence.DeploymentTargetReadyCondition, Status: metav1.ConditionTrue, Reason: "Provisioned"}
	}

	It("leaves a target untouched when its class exists", func() {
		target := newTarget("target", "class.example", ready())
		class := &konfidence.DeploymentClass{ObjectMeta: metav1.ObjectMeta{Name: "class.example"}}
		c := fake.NewClientBuilder().WithScheme(schemeForTests()).WithStatusSubresource(target).WithObjects(target, class).Build()
		r := &Reconciler{Client: c}
		Expect(r.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(target)})).Error().NotTo(HaveOccurred())
		updated := &konfidence.DeploymentTarget{}
		Expect(c.Get(ctx, client.ObjectKeyFromObject(target), updated)).To(Succeed())
		Expect(meta.FindStatusCondition(updated.Status.Conditions, konfidence.DeploymentTargetReadyCondition).Reason).To(Equal("Provisioned"))
	})

	It("marks a new target when its class is missing", func() {
		target := newTarget("target", "missing.example", nil)
		c := fake.NewClientBuilder().WithScheme(schemeForTests()).WithStatusSubresource(target).WithObjects(target).Build()
		Expect((&Reconciler{Client: c}).Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(target)})).Error().NotTo(HaveOccurred())
		updated := &konfidence.DeploymentTarget{}
		Expect(c.Get(ctx, client.ObjectKeyFromObject(target), updated)).To(Succeed())
		condition := meta.FindStatusCondition(updated.Status.Conditions, konfidence.DeploymentTargetReadyCondition)
		Expect(condition.Status).To(Equal(metav1.ConditionFalse))
		Expect(condition.Reason).To(Equal(deploymentClassNotFound))
	})

	It("marks an already-ready target when its class is missing", func() {
		target := newTarget("target", "missing.example", ready())
		c := fake.NewClientBuilder().WithScheme(schemeForTests()).WithStatusSubresource(target).WithObjects(target).Build()
		Expect((&Reconciler{Client: c}).Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(target)})).Error().NotTo(HaveOccurred())
		updated := &konfidence.DeploymentTarget{}
		Expect(c.Get(ctx, client.ObjectKeyFromObject(target), updated)).To(Succeed())
		condition := meta.FindStatusCondition(updated.Status.Conditions, konfidence.DeploymentTargetReadyCondition)
		Expect(condition.Status).To(Equal(metav1.ConditionFalse))
		Expect(condition.Reason).To(Equal(deploymentClassNotFound))
	})
})

var _ = Describe("DeploymentClass deletion watch", Ordered, func() {
	var (
		ctx       context.Context
		k8sClient client.Client
		cancel    context.CancelFunc
		ns        string
	)

	BeforeAll(func() {
		ctx = context.Background()
		k8sClient, cancel = startManager()
		ns = "default"
	})

	AfterAll(func() { cancel() })

	It("reconciles matching targets and ignores unrelated targets", func() {
		class := &konfidence.DeploymentClass{
			ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("class-%d", GinkgoRandomSeed())},
			Spec:       konfidence.DeploymentClassSpec{Controller: "test-controller"},
		}
		otherClass := &konfidence.DeploymentClass{
			ObjectMeta: metav1.ObjectMeta{Name: class.Name + "-other"},
			Spec:       konfidence.DeploymentClassSpec{Controller: "test-controller"},
		}
		Expect(k8sClient.Create(ctx, class)).To(Succeed())
		Expect(k8sClient.Create(ctx, otherClass)).To(Succeed())
		matching := envTarget("matching", ns, class.Name)
		unrelated := envTarget("unrelated", ns, otherClass.Name)
		Expect(k8sClient.Create(ctx, matching)).To(Succeed())
		Expect(k8sClient.Create(ctx, unrelated)).To(Succeed())
		setReady(ctx, k8sClient, matching)
		setReady(ctx, k8sClient, unrelated)
		Expect(k8sClient.Delete(ctx, class)).To(Succeed())

		Eventually(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: matching.Name, Namespace: ns}, matching)).To(Succeed())
			g.Expect(meta.FindStatusCondition(matching.Status.Conditions, konfidence.DeploymentTargetReadyCondition).Reason).To(Equal(deploymentClassNotFound))
		}).Should(Succeed())
		Consistently(func(g Gomega) {
			g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: unrelated.Name, Namespace: ns}, unrelated)).To(Succeed())
			g.Expect(meta.FindStatusCondition(unrelated.Status.Conditions, konfidence.DeploymentTargetReadyCondition).Reason).To(Equal("Provisioned"))
		}).Should(Succeed())
	})
})

func schemeForTests() *runtime.Scheme {
	s := runtime.NewScheme()
	Expect(konfidence.AddToScheme(s)).To(Succeed())
	return s
}

func envTarget(name, namespace, class string) *konfidence.DeploymentTarget {
	return &konfidence.DeploymentTarget{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: konfidence.DeploymentTargetSpec{
			DeploymentClassName: class,
			Connection:          konfidence.DeploymentTargetConnection{Type: "test"},
		},
	}
}

func setReady(ctx context.Context, c client.Client, target *konfidence.DeploymentTarget) {
	// Retry: the cache-backed client's reads lag writes, so Get may miss a fresh Create.
	Eventually(func(g Gomega) {
		g.Expect(c.Get(ctx, client.ObjectKeyFromObject(target), target)).To(Succeed())
		target.Status.Conditions = []metav1.Condition{{
			Type:               konfidence.DeploymentTargetReadyCondition,
			Status:             metav1.ConditionTrue,
			Reason:             "Provisioned",
			LastTransitionTime: metav1.Now(),
		}}
		g.Expect(c.Status().Update(ctx, target)).To(Succeed())
	}).Should(Succeed())

	// wait for the cache to discover the condition
	Eventually(func(g Gomega) {
		g.Expect(c.Get(ctx, client.ObjectKeyFromObject(target), target)).To(Succeed())
		g.Expect(meta.IsStatusConditionTrue(target.Status.Conditions, konfidence.DeploymentTargetReadyCondition)).To(BeTrue())
	}).Should(Succeed())
}

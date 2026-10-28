package deploymenttarget

import (
	"context"

	konfidencev1alpha1 "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("FindReady", func() {
	It("finds ready targets and excludes absent or unready targets", func() {
		scheme := runtime.NewScheme()
		Expect(konfidencev1alpha1.AddToScheme(scheme)).To(Succeed())
		target := readyTarget("prod")
		target.Spec.ArtifactType = "cloud.example.helm"
		unready := namedTarget("unready", 0)
		unready.Spec.ArtifactType = "helm.example.com"
		c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(target, unready).Build()
		got, err := FindReady(context.Background(), c, "landscape", "cloud.example.helm")
		Expect(err).NotTo(HaveOccurred())
		Expect(got.Name).To(Equal(target.Name))
		_, err = FindReady(context.Background(), c, "other", "helm.example.com")
		Expect(err).To(HaveOccurred())
		_, err = FindReady(context.Background(), c, "landscape", "helm.example.com")
		Expect(err).To(HaveOccurred())
	})

	It("rejects multiple ready targets of the same type", func() {
		scheme := runtime.NewScheme()
		Expect(konfidencev1alpha1.AddToScheme(scheme)).To(Succeed())
		objects := []runtime.Object{readyTarget("one"), readyTarget("two")}
		c := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(objects...).Build()
		_, err := FindReady(context.Background(), c, "landscape", "helm.example.com")
		Expect(err).To(MatchError(ContainSubstring("multiple DeploymentTargets")))
	})
})

var _ = Describe("IsReady", func() {
	It("requires a current-generation Ready condition", func() {
		target := readyTarget("prod")
		target.Generation = 3
		Expect(IsReady(target)).To(BeFalse())
		target.Status.Conditions[0].ObservedGeneration = target.Generation
		Expect(IsReady(target)).To(BeTrue())
	})
})

func readyTarget(name string) *konfidencev1alpha1.DeploymentTarget {
	return &konfidencev1alpha1.DeploymentTarget{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "landscape"},
		Spec:       konfidencev1alpha1.DeploymentTargetSpec{ArtifactType: "helm.example.com"},
		Status: konfidencev1alpha1.DeploymentTargetStatus{Conditions: []metav1.Condition{{
			Type: konfidencev1alpha1.DeploymentTargetReadyCondition, Status: metav1.ConditionTrue,
		}}},
	}
}

func namedTarget(name string, generation int64) *konfidencev1alpha1.DeploymentTarget {
	return &konfidencev1alpha1.DeploymentTarget{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "landscape", Generation: generation},
		Spec:       konfidencev1alpha1.DeploymentTargetSpec{ArtifactType: "cloud.example.helm"},
	}
}

package deploymenttarget

import (
	"context"
	"errors"

	konfidencev1alpha1 "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("reconciler", func() {
	It("sets the Ready condition with the current generation", func() {
		scheme := runtime.NewScheme()
		Expect(konfidencev1alpha1.AddToScheme(scheme)).To(Succeed())
		dt := namedTarget("prod", 4)
		c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(dt).WithObjects(dt).Build()
		r := NewControllerManagedBy(nil).For(konfidencev1alpha1.ArtifactTypeSet{"cloud.example.helm": {}}).
			Named("example.com/deployer:v1.2.3").build(c, nil, "example.com/deployer:v1.2.3")
		original := dt.DeepCopy()
		r.setReady(dt, metav1.ConditionTrue, ReasonAccepted, "valid target")
		Expect(c.Status().Patch(context.Background(), dt, client.MergeFrom(original))).To(Succeed())
		updated := &konfidencev1alpha1.DeploymentTarget{}
		Expect(c.Get(context.Background(), client.ObjectKeyFromObject(dt), updated)).To(Succeed())
		condition := meta.FindStatusCondition(updated.Status.Conditions, konfidencev1alpha1.DeploymentTargetReadyCondition)
		Expect(condition).NotTo(BeNil())
		Expect(condition.Status).To(Equal(metav1.ConditionTrue))
		Expect(condition.ObservedGeneration).To(Equal(int64(4)))
	})

	It("records controller identity and validation outcomes", func() {
		ctx := context.Background()
		scheme := runtime.NewScheme()
		Expect(konfidencev1alpha1.AddToScheme(scheme)).To(Succeed())
		valid := namedTarget("valid", 2)
		invalid := namedTarget("invalid", 3)
		unsupported := namedTarget("unsupported", 0)
		unsupported.Spec.ArtifactType = "other.example.com"
		c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(&konfidencev1alpha1.DeploymentTarget{}).WithObjects(valid, invalid, unsupported).Build()
		r := NewControllerManagedBy(nil).For(konfidencev1alpha1.ArtifactTypeSet{"cloud.example.helm": {}}).
			WithValidation(func(_ context.Context, _ client.Reader, dt *konfidencev1alpha1.DeploymentTarget) (ValidationResult, error) {
				if dt.Name == invalid.Name {
					return ValidationFailure("invalid parameters"), nil
				}
				return ValidationSuccess("parameters accepted"), nil
			}).build(c, nil, defaultControllerName)
		for _, tc := range []struct {
			name, reason string
			status       metav1.ConditionStatus
			generation   int64
		}{{valid.Name, ReasonAccepted, metav1.ConditionTrue, 2}, {invalid.Name, ReasonInvalidParameters, metav1.ConditionFalse, 3}} {
			_, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKey{Namespace: "landscape", Name: tc.name}})
			Expect(err).NotTo(HaveOccurred())
			got := &konfidencev1alpha1.DeploymentTarget{}
			Expect(c.Get(ctx, client.ObjectKey{Namespace: "landscape", Name: tc.name}, got)).To(Succeed())
			condition := meta.FindStatusCondition(got.Status.Conditions, konfidencev1alpha1.DeploymentTargetReadyCondition)
			Expect(got.Status.Controller).To(Equal(defaultControllerName))
			Expect(condition).NotTo(BeNil())
			Expect(condition.Status).To(Equal(tc.status))
			Expect(condition.Reason).To(Equal(tc.reason))
			Expect(condition.ObservedGeneration).To(Equal(tc.generation))
		}
		_, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(unsupported)})
		Expect(err).NotTo(HaveOccurred())
		got := &konfidencev1alpha1.DeploymentTarget{}
		Expect(c.Get(ctx, client.ObjectKeyFromObject(unsupported), got)).To(Succeed())
		Expect(got.Status.Controller).To(BeEmpty())
		Expect(got.Status.Conditions).To(BeEmpty())
	})

	It("updates status before returning a validation error", func() {
		scheme := runtime.NewScheme()
		Expect(konfidencev1alpha1.AddToScheme(scheme)).To(Succeed())
		target := namedTarget("transient", 1)
		c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(target).WithObjects(target).Build()
		validationErr := errors.New("temporary read failure")
		r := NewControllerManagedBy(nil).For(konfidencev1alpha1.ArtifactTypeSet{"cloud.example.helm": {}}).
			WithValidation(func(context.Context, client.Reader, *konfidencev1alpha1.DeploymentTarget) (ValidationResult, error) {
				return nil, validationErr
			}).build(c, nil, defaultControllerName)
		_, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: client.ObjectKeyFromObject(target)})
		Expect(err).To(MatchError(validationErr))
		got := &konfidencev1alpha1.DeploymentTarget{}
		Expect(c.Get(context.Background(), client.ObjectKeyFromObject(target), got)).To(Succeed())
		condition := meta.FindStatusCondition(got.Status.Conditions, konfidencev1alpha1.DeploymentTargetReadyCondition)
		Expect(condition).NotTo(BeNil())
		Expect(condition.Status).To(Equal(metav1.ConditionFalse))
		Expect(condition.Reason).To(Equal(ReasonValidationError))
		Expect(condition.Message).To(Equal(validationErr.Error()))
	})
})

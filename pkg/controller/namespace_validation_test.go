package utils_test

import (
	"context"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	utils "github.com/konfidence-project/konfidence/pkg/controller"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Namespace guards", func() {
	var (
		ctx    context.Context
		scheme *runtime.Scheme
	)

	BeforeEach(func() {
		ctx = context.Background()
		scheme = runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
		Expect(konfidence.AddToScheme(scheme)).To(Succeed())
	})

	newStage := func(namespace string) *konfidence.Stage {
		return &konfidence.Stage{ObjectMeta: metav1.ObjectMeta{Name: "guarded", Namespace: namespace}}
	}

	It("continues (nil result) when the namespace has the required type label", func() {
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
			Name:   "landscape-ns",
			Labels: map[string]string{utils.ProjectTypeLabel: "landscape"},
		}}
		stage := newStage(ns.Name)
		c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(stage).WithObjects(ns, stage).Build()

		res, err := utils.MustBeInLandscapeNamespace(ctx, c, stage, &stage.Status.Conditions, konfidence.StageReady)
		Expect(err).NotTo(HaveOccurred())
		Expect(res).To(BeNil())
	})

	It("requeues and sets the condition (naming the missing label) when the type label is absent", func() {
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "plain-ns"}}
		stage := newStage(ns.Name)
		c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(stage).WithObjects(ns, stage).Build()

		res, err := utils.MustBeInLandscapeNamespace(ctx, c, stage, &stage.Status.Conditions, konfidence.StageReady)
		Expect(err).NotTo(HaveOccurred())
		Expect(res).NotTo(BeNil())
		Expect(res.RequeueAfter).To(Equal(utils.NamespaceNotReadyRequeueInterval))

		cond := meta.FindStatusCondition(stage.Status.Conditions, konfidence.StageReady)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Status).To(Equal(metav1.ConditionFalse))
		Expect(cond.Reason).To(Equal(utils.NamespaceNotReadyReason))
		Expect(cond.Message).To(ContainSubstring("must be created in a landscape namespace"))
		Expect(cond.Message).To(ContainSubstring(utils.ProjectTypeLabel))
	})

	It("requeues when the type label has the wrong value", func() {
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
			Name:   "project-ns",
			Labels: map[string]string{utils.ProjectTypeLabel: "project"},
		}}
		stage := newStage(ns.Name)
		c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(stage).WithObjects(ns, stage).Build()

		res, err := utils.MustBeInLandscapeNamespace(ctx, c, stage, &stage.Status.Conditions, konfidence.StageReady)
		Expect(err).NotTo(HaveOccurred())
		Expect(res).NotTo(BeNil())
		cond := meta.FindStatusCondition(stage.Status.Conditions, konfidence.StageReady)
		Expect(cond).NotTo(BeNil())
		Expect(cond.Reason).To(Equal(utils.NamespaceNotReadyReason))
	})

	It("is a no-op (nil result) when the object is being deleted", func() {
		stage := newStage("plain-ns")
		now := metav1.NewTime(time.Now())
		stage.DeletionTimestamp = &now
		c := fake.NewClientBuilder().WithScheme(scheme).Build()

		res, err := utils.MustBeInLandscapeNamespace(ctx, c, stage, &stage.Status.Conditions, konfidence.StageReady)
		Expect(err).NotTo(HaveOccurred())
		Expect(res).To(BeNil())
	})

	It("checks for a project namespace via MustBeInProjectNamespace", func() {
		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
			Name:   "project-ns",
			Labels: map[string]string{utils.ProjectTypeLabel: "project"},
		}}
		vt := &konfidence.VectorTemplate{ObjectMeta: metav1.ObjectMeta{Name: "vt", Namespace: ns.Name}}
		c := fake.NewClientBuilder().WithScheme(scheme).WithStatusSubresource(vt).WithObjects(ns, vt).Build()

		res, err := utils.MustBeInProjectNamespace(ctx, c, vt, &vt.Status.Conditions, konfidence.VectorTemplateReadyCondition)
		Expect(err).NotTo(HaveOccurred())
		Expect(res).To(BeNil())
	})
})

package webhook_test

import (
	"context"

	utils "github.com/konfidence-project/konfidence/pkg/controller"
	"github.com/konfidence-project/konfidence/pkg/webhook"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

//nolint:dupl // Tests intentionally duplicated to cover both project and landscape namespace validation
var _ = Describe("Webhook Validation", func() {
	var (
		ctx        context.Context
		fakeClient client.Client
		scheme     *runtime.Scheme
	)

	BeforeEach(func() {
		ctx = context.Background()
		scheme = runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
	})

	Describe("ValidateProjectNamespace", func() {
		Context("when namespace is valid", func() {
			It("should return no error", func() {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-namespace",
						Labels: map[string]string{
							utils.ProjectTypeLabel: "project",
							utils.ProjectNameLabel: "test-project",
						},
					},
				}
				fakeClient = fake.NewClientBuilder().WithScheme(scheme).WithObjects(ns).Build()

				err := webhook.ValidateProjectNamespace(ctx, fakeClient, "test-namespace")

				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when namespace does not exist", func() {
			It("should return an error", func() {
				fakeClient = fake.NewClientBuilder().WithScheme(scheme).Build()

				err := webhook.ValidateProjectNamespace(ctx, fakeClient, "nonexistent-namespace")

				Expect(err).To(HaveOccurred())
			})
		})

		Context("when namespace has no labels", func() {
			It("should return an error", func() {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-namespace",
					},
				}
				fakeClient = fake.NewClientBuilder().WithScheme(scheme).WithObjects(ns).Build()

				err := webhook.ValidateProjectNamespace(ctx, fakeClient, "test-namespace")

				Expect(err).To(HaveOccurred())
			})
		})

		Context("when namespace has wrong type label", func() {
			It("should return an error", func() {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-namespace",
						Labels: map[string]string{
							utils.ProjectTypeLabel: "landscape",
							utils.ProjectNameLabel: "test-project",
						},
					},
				}
				fakeClient = fake.NewClientBuilder().WithScheme(scheme).WithObjects(ns).Build()

				err := webhook.ValidateProjectNamespace(ctx, fakeClient, "test-namespace")

				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("ValidateLandscapeNamespace", func() {
		Context("when namespace is valid", func() {
			It("should return no error", func() {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-namespace",
						Labels: map[string]string{
							utils.ProjectTypeLabel:   "landscape",
							utils.LandscapeNameLabel: "test-landscape",
						},
					},
				}
				fakeClient = fake.NewClientBuilder().WithScheme(scheme).WithObjects(ns).Build()

				err := webhook.ValidateLandscapeNamespace(ctx, fakeClient, "test-namespace")

				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when namespace does not exist", func() {
			It("should return an error", func() {
				fakeClient = fake.NewClientBuilder().WithScheme(scheme).Build()

				err := webhook.ValidateLandscapeNamespace(ctx, fakeClient, "nonexistent-namespace")

				Expect(err).To(HaveOccurred())
			})
		})

		Context("when namespace has no labels", func() {
			It("should return an error", func() {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-namespace",
					},
				}
				fakeClient = fake.NewClientBuilder().WithScheme(scheme).WithObjects(ns).Build()

				err := webhook.ValidateLandscapeNamespace(ctx, fakeClient, "test-namespace")

				Expect(err).To(HaveOccurred())
			})
		})

		Context("when namespace has wrong type label", func() {
			It("should return an error", func() {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-namespace",
						Labels: map[string]string{
							utils.ProjectTypeLabel:   "project",
							utils.LandscapeNameLabel: "test-landscape",
						},
					},
				}
				fakeClient = fake.NewClientBuilder().WithScheme(scheme).WithObjects(ns).Build()

				err := webhook.ValidateLandscapeNamespace(ctx, fakeClient, "test-namespace")

				Expect(err).To(HaveOccurred())
			})
		})
	})
})

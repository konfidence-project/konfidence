package v1alpha1

import (
	"context"

	pkgctrl "github.com/konfidence-project/konfidence/pkg/controller"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("LandscapeValidator", func() {
	var (
		ctx        context.Context
		validator  *LandscapeValidator
		fakeClient client.Client
	)

	BeforeEach(func() {
		ctx = context.Background()
		scheme := runtime.NewScheme()
		Expect(AddToScheme(scheme)).To(Succeed())
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
		fakeClient = fake.NewClientBuilder().WithScheme(scheme).Build()
		validator = &LandscapeValidator{Client: fakeClient}
	})

	Describe("ValidateCreate", func() {
		Context("when namespace has correct project labels", func() {
			It("should succeed", func() {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-project-ns",
						Labels: map[string]string{
							pkgctrl.ProjectTypeLabel: "project",
							pkgctrl.ProjectNameLabel: "test-project",
						},
					},
				}
				Expect(fakeClient.Create(ctx, ns)).To(Succeed())

				landscape := &Landscape{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-landscape",
						Namespace: "test-project-ns",
					},
				}

				warnings, err := validator.ValidateCreate(ctx, landscape)
				Expect(err).NotTo(HaveOccurred())
				Expect(warnings).To(BeEmpty())
			})
		})

		Context("when namespace is missing project labels", func() {
			It("should fail with field error", func() {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "regular-ns",
					},
				}
				Expect(fakeClient.Create(ctx, ns)).To(Succeed())

				landscape := &Landscape{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-landscape",
						Namespace: "regular-ns",
					},
				}

				warnings, err := validator.ValidateCreate(ctx, landscape)
				Expect(warnings).To(BeEmpty())
				Expect(err).To(HaveOccurred())

				fieldErr, ok := err.(*field.Error)
				Expect(ok).To(BeTrue(), "error should be a field.Error")
				Expect(fieldErr.Field).To(Equal("metadata.namespace"))
				Expect(fieldErr.BadValue).To(Equal("regular-ns"))
			})
		})

		Context("when namespace is missing project-name label", func() {
			It("should fail with field error", func() {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "incomplete-project-ns",
						Labels: map[string]string{
							pkgctrl.ProjectTypeLabel: "project",
							// Missing ProjectNameLabel
						},
					},
				}
				Expect(fakeClient.Create(ctx, ns)).To(Succeed())

				landscape := &Landscape{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-landscape",
						Namespace: "incomplete-project-ns",
					},
				}

				warnings, err := validator.ValidateCreate(ctx, landscape)
				Expect(warnings).To(BeEmpty())
				Expect(err).To(HaveOccurred())

				fieldErr, ok := err.(*field.Error)
				Expect(ok).To(BeTrue(), "error should be a field.Error")
				Expect(fieldErr.Field).To(Equal("metadata.namespace"))
				Expect(fieldErr.BadValue).To(Equal("incomplete-project-ns"))
			})
		})
	})

	Describe("ValidateUpdate", func() {
		Context("when namespace has correct project labels", func() {
			It("should succeed", func() {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-project-ns",
						Labels: map[string]string{
							pkgctrl.ProjectTypeLabel: "project",
							pkgctrl.ProjectNameLabel: "test-project",
						},
					},
				}
				Expect(fakeClient.Create(ctx, ns)).To(Succeed())

				oldLandscape := &Landscape{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-landscape",
						Namespace: "test-project-ns",
					},
				}
				newLandscape := oldLandscape.DeepCopy()
				newLandscape.Spec.DisplayName = "Updated Display Name"

				warnings, err := validator.ValidateUpdate(ctx, oldLandscape, newLandscape)
				Expect(err).NotTo(HaveOccurred())
				Expect(warnings).To(BeEmpty())
			})
		})

		Context("when namespace is missing project labels", func() {
			It("should fail with field error", func() {
				ns := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: "regular-ns",
					},
				}
				Expect(fakeClient.Create(ctx, ns)).To(Succeed())

				oldLandscape := &Landscape{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-landscape",
						Namespace: "regular-ns",
					},
				}
				newLandscape := oldLandscape.DeepCopy()

				warnings, err := validator.ValidateUpdate(ctx, oldLandscape, newLandscape)
				Expect(warnings).To(BeEmpty())
				Expect(err).To(HaveOccurred())

				fieldErr, ok := err.(*field.Error)
				Expect(ok).To(BeTrue(), "error should be a field.Error")
				Expect(fieldErr.Field).To(Equal("metadata.namespace"))
			})
		})
	})

	Describe("ValidateDelete", func() {
		It("should always succeed", func() {
			landscape := &Landscape{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-landscape",
					Namespace: "any-namespace",
				},
			}

			warnings, err := validator.ValidateDelete(ctx, landscape)
			Expect(err).NotTo(HaveOccurred())
			Expect(warnings).To(BeEmpty())
		})
	})
})

package v1alpha1

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("DeploymentClassValidator", func() {
	var (
		ctx       context.Context
		validator *DeploymentClassValidator
		scheme    *runtime.Scheme
	)

	BeforeEach(func() {
		ctx = context.Background()
		scheme = runtime.NewScheme()
		Expect(AddToScheme(scheme)).To(Succeed())
	})

	Describe("ValidateCreate", func() {
		Context("when no other DeploymentClass exists", func() {
			It("should allow creation", func() {
				fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
				validator = &DeploymentClassValidator{Client: fakeClient}

				dc := &DeploymentClass{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubernetes-helm-deployer",
					},
					Spec: DeploymentClassSpec{
						Type:       "konfidence.cloud/helm",
						Controller: "kubernetes-landscape-orchestrator",
					},
				}

				warnings, err := validator.ValidateCreate(ctx, dc)
				Expect(err).NotTo(HaveOccurred())
				Expect(warnings).To(BeEmpty())
			})
		})

		Context("when another DeploymentClass with different type exists", func() {
			It("should allow creation", func() {
				existing := &DeploymentClass{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubernetes-kustomize-deployer",
					},
					Spec: DeploymentClassSpec{
						Type:       "konfidence.cloud/kustomize",
						Controller: "kubernetes-landscape-orchestrator",
					},
				}

				fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()
				validator = &DeploymentClassValidator{Client: fakeClient}

				dc := &DeploymentClass{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubernetes-helm-deployer",
					},
					Spec: DeploymentClassSpec{
						Type:       "konfidence.cloud/helm",
						Controller: "kubernetes-landscape-orchestrator",
					},
				}

				warnings, err := validator.ValidateCreate(ctx, dc)
				Expect(err).NotTo(HaveOccurred())
				Expect(warnings).To(BeEmpty())
			})
		})

		Context("when another DeploymentClass with same type exists", func() {
			It("should reject creation", func() {
				existing := &DeploymentClass{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubernetes-helm-deployer",
					},
					Spec: DeploymentClassSpec{
						Type:       "konfidence.cloud/helm",
						Controller: "kubernetes-landscape-orchestrator",
					},
				}

				fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()
				validator = &DeploymentClassValidator{Client: fakeClient}

				dc := &DeploymentClass{
					ObjectMeta: metav1.ObjectMeta{
						Name: "another-helm-deployer",
					},
					Spec: DeploymentClassSpec{
						Type:       "konfidence.cloud/helm",
						Controller: "another-controller",
					},
				}

				warnings, err := validator.ValidateCreate(ctx, dc)
				Expect(warnings).To(BeEmpty())
				Expect(err).To(HaveOccurred())

				fieldErr, ok := err.(*field.Error)
				Expect(ok).To(BeTrue(), "error should be a field.Error")
				Expect(fieldErr.Field).To(Equal("spec.type"))
				Expect(fieldErr.BadValue).To(Equal("konfidence.cloud/helm"))
			})
		})
	})

	Describe("ValidateUpdate", func() {
		Context("when type hasn't changed", func() {
			It("should allow update", func() {
				existing := &DeploymentClass{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubernetes-helm-deployer",
					},
					Spec: DeploymentClassSpec{
						Type:       "konfidence.cloud/helm",
						Controller: "kubernetes-landscape-orchestrator",
					},
				}

				fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()
				validator = &DeploymentClassValidator{Client: fakeClient}

				oldDC := existing.DeepCopy()
				newDC := existing.DeepCopy()
				newDC.Spec.Controller = "updated-controller"

				warnings, err := validator.ValidateUpdate(ctx, oldDC, newDC)
				Expect(err).NotTo(HaveOccurred())
				Expect(warnings).To(BeEmpty())
			})
		})

		Context("when type changed to a new unique value", func() {
			It("should allow update", func() {
				existing := &DeploymentClass{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubernetes-helm-deployer",
					},
					Spec: DeploymentClassSpec{
						Type:       "konfidence.cloud/helm",
						Controller: "kubernetes-landscape-orchestrator",
					},
				}

				fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()
				validator = &DeploymentClassValidator{Client: fakeClient}

				oldDC := existing.DeepCopy()
				newDC := existing.DeepCopy()
				newDC.Spec.Type = "konfidence.cloud/helm-v2"

				warnings, err := validator.ValidateUpdate(ctx, oldDC, newDC)
				Expect(err).NotTo(HaveOccurred())
				Expect(warnings).To(BeEmpty())
			})
		})

		Context("when type changed to an already used value", func() {
			It("should reject update", func() {
				existing1 := &DeploymentClass{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubernetes-helm-deployer",
					},
					Spec: DeploymentClassSpec{
						Type:       "konfidence.cloud/helm",
						Controller: "kubernetes-landscape-orchestrator",
					},
				}
				existing2 := &DeploymentClass{
					ObjectMeta: metav1.ObjectMeta{
						Name: "kubernetes-kustomize-deployer",
					},
					Spec: DeploymentClassSpec{
						Type:       "konfidence.cloud/kustomize",
						Controller: "kubernetes-landscape-orchestrator",
					},
				}

				fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing1, existing2).Build()
				validator = &DeploymentClassValidator{Client: fakeClient}

				oldDC := existing1.DeepCopy()
				newDC := existing1.DeepCopy()
				newDC.Spec.Type = "konfidence.cloud/kustomize"

				warnings, err := validator.ValidateUpdate(ctx, oldDC, newDC)
				Expect(warnings).To(BeEmpty())
				Expect(err).To(HaveOccurred())

				fieldErr, ok := err.(*field.Error)
				Expect(ok).To(BeTrue(), "error should be a field.Error")
				Expect(fieldErr.Field).To(Equal("spec.type"))
			})
		})
	})

	Describe("ValidateDelete", func() {
		It("should always allow deletion", func() {
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
			validator = &DeploymentClassValidator{Client: fakeClient}

			dc := &DeploymentClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "kubernetes-helm-deployer",
				},
				Spec: DeploymentClassSpec{
					Type:       "konfidence.cloud/helm",
					Controller: "kubernetes-landscape-orchestrator",
				},
			}

			warnings, err := validator.ValidateDelete(ctx, dc)
			Expect(err).NotTo(HaveOccurred())
			Expect(warnings).To(BeEmpty())
		})
	})
})

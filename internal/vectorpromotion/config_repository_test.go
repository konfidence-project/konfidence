package vectorpromotion

import (
	"context"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("ConfigRepository", func() {
	var ctx context.Context

	newConfig := func(name, namespace string) *konfidence.VectorPromotionConfig {
		return &konfidence.VectorPromotionConfig{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
			Spec: konfidence.VectorPromotionConfigSpec{
				Source: konfidence.PromotionSourceReference{Kind: "Stage", Name: "src", Landscape: "dev"},
				Target: konfidence.PromotionTargetReference{Kind: "Stage", Name: "dst", Landscape: "prod"},
			},
		}
	}

	newConfigRepository := func(objects ...client.Object) ConfigRepository {
		testScheme := runtime.NewScheme()
		Expect(scheme.AddToScheme(testScheme)).To(Succeed())
		Expect(konfidence.AddToScheme(testScheme)).To(Succeed())
		c := fake.NewClientBuilder().
			WithScheme(testScheme).
			WithObjects(objects...).
			Build()
		return NewConfigRepository(c)
	}

	BeforeEach(func() {
		ctx = context.Background()
	})

	Describe("Get", func() {
		It("returns the config for a matching namespace and name", func() {
			repo := newConfigRepository(newConfig("cfg", "default"))

			got, err := repo.Get(ctx, "default", "cfg")
			Expect(err).NotTo(HaveOccurred())
			Expect(got.Name).To(Equal("cfg"))
			Expect(got.Namespace).To(Equal("default"))
			Expect(got.Spec.Source.Name).To(Equal("src"))
			Expect(got.Spec.Target.Name).To(Equal("dst"))
		})

		It("returns ErrVectorPromotionConfigNotFound for a missing name", func() {
			repo := newConfigRepository()

			_, err := repo.Get(ctx, "default", "missing")
			Expect(err).To(MatchError(ErrVectorPromotionConfigNotFound))
		})

		It("does not return a config that exists only in a different namespace", func() {
			repo := newConfigRepository(newConfig("cfg", "other-namespace"))

			_, err := repo.Get(ctx, "default", "cfg")
			Expect(err).To(MatchError(ErrVectorPromotionConfigNotFound))
		})
	})

	Describe("List", func() {
		It("returns only configs in the requested namespace", func() {
			inNamespace := newConfig("cfg-a", "default")
			otherNamespace := newConfig("cfg-b", "other-namespace")
			repo := newConfigRepository(inNamespace, otherNamespace)

			result, err := repo.List(ctx, "default")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].Name).To(Equal("cfg-a"))
		})

		It("returns empty when the namespace has no configs", func() {
			repo := newConfigRepository()

			result, err := repo.List(ctx, "default")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeEmpty())
		})
	})
})

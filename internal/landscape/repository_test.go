package landscape_test

import (
	"context"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/landscape"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newScheme() *runtime.Scheme {
	GinkgoHelper()
	scheme := runtime.NewScheme()
	Expect(clientgoscheme.AddToScheme(scheme)).To(Succeed())
	Expect(konfidence.AddToScheme(scheme)).To(Succeed())
	return scheme
}

func fakeClient(objs ...client.Object) client.Client {
	GinkgoHelper()
	return fake.NewClientBuilder().WithScheme(newScheme()).WithObjects(objs...).Build()
}

func landscapeFixture(name, namespace, displayName string) *konfidence.Landscape {
	return &konfidence.Landscape{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec:       konfidence.LandscapeSpec{DisplayName: displayName},
	}
}

func scopedLandscapeFixture(name, namespace, managedNamespace string) *konfidence.Landscape {
	l := landscapeFixture(name, namespace, name)
	l.Status = konfidence.LandscapeStatus{Namespace: managedNamespace}
	return l
}

var _ = Describe("Repository", func() {
	Describe("ListForProject", func() {
		It("returns landscapes", func() {
			repository := landscape.NewRepository(fakeClient(
				landscapeFixture("dev", "kden-p-my-project", "Dev"),
				landscapeFixture("staging", "kden-p-my-project", "Staging"),
			))

			result, err := repository.ListForProject(context.Background(), "kden-p-my-project")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(HaveLen(2))
		})

		It("returns an empty list for an empty namespace", func() {
			result, err := landscape.NewRepository(fakeClient()).ListForProject(context.Background(), "kden-p-empty")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(BeEmpty())
		})

		It("only returns landscapes from the requested namespace", func() {
			repository := landscape.NewRepository(fakeClient(
				landscapeFixture("dev", "kden-p-project-a", "Dev A"),
				landscapeFixture("dev", "kden-p-project-b", "Dev B"),
			))

			result, err := repository.ListForProject(context.Background(), "kden-p-project-a")
			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(ConsistOf(HaveField("Name", "dev")))
		})
	})

	It("gets a landscape", func() {
		want := landscapeFixture("dev", "kden-p-my-project", "Development")

		got, err := landscape.NewRepository(fakeClient(want)).Get(context.Background(), "kden-p-my-project", "dev")
		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(And(
			HaveField("Name", want.Name),
			HaveField("Spec.DisplayName", want.Spec.DisplayName),
		))
	})

	Describe("ResolveScope", func() {
		It("returns all landscapes of a project", func() {
			repository := landscape.NewRepository(fakeClient(
				scopedLandscapeFixture("dev", "kden-p-my-project", "kden-l-dev-1234"),
				scopedLandscapeFixture("staging", "kden-p-my-project", "kden-l-staging-5678"),
			))

			scope, err := repository.ResolveScope(context.Background(), "kden-p-my-project")
			Expect(err).NotTo(HaveOccurred())
			Expect(scope).To(ConsistOf(
				And(HaveField("Landscape.Name", "dev"), HaveField("Namespace", "kden-l-dev-1234")),
				And(HaveField("Landscape.Name", "staging"), HaveField("Namespace", "kden-l-staging-5678")),
			))
		})

		It("narrows the scope by landscape ID", func() {
			repository := landscape.NewRepository(fakeClient(
				scopedLandscapeFixture("dev", "kden-p-my-project", "kden-l-dev-1234"),
				scopedLandscapeFixture("staging", "kden-p-my-project", "kden-l-staging-5678"),
			))

			scope, err := repository.ResolveScope(context.Background(), "kden-p-my-project", landscape.WithLandscapeId("staging"))
			Expect(err).NotTo(HaveOccurred())
			Expect(scope).To(ConsistOf(And(
				HaveField("Landscape.Name", "staging"),
				HaveField("Namespace", "kden-l-staging-5678"),
			)))
		})

		DescribeTable("rejects an unknown landscape ID",
			func(id string) {
				repository := landscape.NewRepository(fakeClient(
					scopedLandscapeFixture("dev", "kden-p-my-project", "kden-l-dev-1234"),
				))

				scope, err := repository.ResolveScope(context.Background(), "kden-p-my-project", landscape.WithLandscapeId(id))
				Expect(err).To(MatchError(landscape.ErrLandscapeNotFound))
				Expect(scope).To(BeNil())
			},
			Entry("when it does not exist", "nope"),
			Entry("when it is empty", ""),
		)

		It("keeps a provisioning landscape in scope", func() {
			repository := landscape.NewRepository(fakeClient(
				scopedLandscapeFixture("dev", "kden-p-my-project", ""),
			))

			scope, err := repository.ResolveScope(context.Background(), "kden-p-my-project", landscape.WithLandscapeId("dev"))
			Expect(err).NotTo(HaveOccurred())
			Expect(scope).To(ConsistOf(HaveField("Namespace", "")))
		})

		It("excludes other projects", func() {
			repository := landscape.NewRepository(fakeClient(
				scopedLandscapeFixture("dev", "kden-p-project-a", "kden-l-dev-a"),
				scopedLandscapeFixture("dev", "kden-p-project-b", "kden-l-dev-b"),
			))

			scope, err := repository.ResolveScope(context.Background(), "kden-p-project-a")
			Expect(err).NotTo(HaveOccurred())
			Expect(scope).To(ConsistOf(HaveField("Namespace", "kden-l-dev-a")))
		})
	})
})

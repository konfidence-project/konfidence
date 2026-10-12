package project_test

import (
	"context"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/auth"
	"github.com/konfidence-project/konfidence/internal/project"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func projectScheme() *runtime.Scheme {
	GinkgoHelper()
	scheme := runtime.NewScheme()
	Expect(konfidence.AddToScheme(scheme)).To(Succeed())
	return scheme
}

var _ = Describe("Repository", func() {
	It("gets a project", func() {
		want := &konfidence.Project{ObjectMeta: metav1.ObjectMeta{Name: "my-project"}}
		k8s := fake.NewClientBuilder().WithScheme(projectScheme()).WithObjects(want).Build()

		got, err := project.NewRepository(k8s).Get(context.Background(), want.Name)
		Expect(err).NotTo(HaveOccurred())
		Expect(got.Name).To(Equal(want.Name))
	})

	It("returns not found when the project does not exist", func() {
		k8s := fake.NewClientBuilder().WithScheme(projectScheme()).Build()

		_, err := project.NewRepository(k8s).Get(context.Background(), "missing")

		Expect(err).To(MatchError(project.ErrNotFound))
	})

	It("lists only authorized projects", func() {
		k8s := fake.NewClientBuilder().WithScheme(projectScheme()).WithObjects(
			&konfidence.Project{ObjectMeta: metav1.ObjectMeta{Name: "one"}},
			&konfidence.Project{ObjectMeta: metav1.ObjectMeta{Name: "two"}},
		).Build()

		projects, err := project.NewRepository(k8s).List(context.Background(), auth.ProjectRoles{
			"one": {"admin"},
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(projects).To(ConsistOf(HaveField("Name", "one")))
	})
})

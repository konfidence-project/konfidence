package handler

import (
	"context"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/api/openapi"
	"github.com/konfidence-project/konfidence/internal/api/session"
	"github.com/konfidence-project/konfidence/internal/auth"
	landscapedomain "github.com/konfidence-project/konfidence/internal/landscape"
	projectdomain "github.com/konfidence-project/konfidence/internal/project"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newTestScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(s)
	_ = konfidence.AddToScheme(s)
	return s
}

func fakeK8s(objs ...client.Object) client.Client {
	return fake.NewClientBuilder().
		WithScheme(newTestScheme()).
		WithObjects(objs...).
		WithStatusSubresource(&konfidence.Project{}).
		Build()
}

func landscapeProjectFixture(name, namespace string) *konfidence.Project {
	return &konfidence.Project{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status:     konfidence.ProjectStatus{Namespace: namespace},
	}
}

func landscapeFixture(name, namespace, displayName string) *konfidence.Landscape {
	return &konfidence.Landscape{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec:       konfidence.LandscapeSpec{DisplayName: displayName},
	}
}

func projectHandlerWith(objs ...client.Object) *projectHandler {
	k8s := fakeK8s(objs...)
	return newProjectHandler(
		projectdomain.NewRepository(k8s),
		landscapedomain.NewRepository(k8s),
	)
}

func ctxWithProjectRoles(roles auth.ProjectRoles) context.Context {
	return session.NewContext(context.Background(), &session.Session{
		Context: session.Context{ProjectRoles: roles},
	})
}

var _ = Describe("ListLandscapesV1", func() {
	It("returns all landscapes for a project", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		l1 := landscapeFixture("dev", "kden-p-my-project", "Dev")
		l2 := landscapeFixture("staging", "kden-p-my-project", "Staging")
		h := projectHandlerWith(project, l1, l2)

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"my-project": {"admin"}})
		resp, err := h.ListLandscapesV1(ctx, openapi.ListLandscapesV1RequestObject{ProjectId: "my-project"})
		Expect(err).NotTo(HaveOccurred())

		ok, is200 := resp.(openapi.ListLandscapesV1200JSONResponse)
		Expect(is200).To(BeTrue())
		Expect(ok.Data).To(HaveLen(2))
		Expect([]string{ok.Data[0].Id, ok.Data[1].Id}).To(ConsistOf("dev", "staging"))
	})

	It("returns an empty list when project has no landscapes", func() {
		project := landscapeProjectFixture("empty-project", "kden-p-empty-project")
		h := projectHandlerWith(project)

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"empty-project": {"admin"}})
		resp, err := h.ListLandscapesV1(ctx, openapi.ListLandscapesV1RequestObject{ProjectId: "empty-project"})
		Expect(err).NotTo(HaveOccurred())

		ok := resp.(openapi.ListLandscapesV1200JSONResponse)
		Expect(ok.Data).To(BeEmpty())
	})

	It("maps landscape fields correctly", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		l := landscapeFixture("dev", "kden-p-my-project", "Development")
		h := projectHandlerWith(project, l)

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"my-project": {"admin"}})
		resp, err := h.ListLandscapesV1(ctx, openapi.ListLandscapesV1RequestObject{ProjectId: "my-project"})
		Expect(err).NotTo(HaveOccurred())

		item := resp.(openapi.ListLandscapesV1200JSONResponse).Data[0]
		Expect(item.Id).To(Equal("dev"))
		Expect(item.Name).To(Equal("Development"))
	})

	It("returns 404 when project does not exist", func() {
		h := projectHandlerWith()

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"nonexistent": {"admin"}})
		resp, err := h.ListLandscapesV1(ctx, openapi.ListLandscapesV1RequestObject{ProjectId: "nonexistent"})
		Expect(err).NotTo(HaveOccurred())

		_, is404 := resp.(openapi.ListLandscapesV1404JSONResponse)
		Expect(is404).To(BeTrue())
	})

	It("returns 401 when no session is present", func() {
		project := landscapeProjectFixture("my-project", "kden-p-my-project")
		h := projectHandlerWith(project)

		resp, err := h.ListLandscapesV1(context.Background(), openapi.ListLandscapesV1RequestObject{ProjectId: "my-project"})
		Expect(err).NotTo(HaveOccurred())

		_, is401 := resp.(openapi.ListLandscapesV1401JSONResponse)
		Expect(is401).To(BeTrue())
	})

	It("only returns landscapes belonging to the requested project", func() {
		project := landscapeProjectFixture("project-a", "kden-p-project-a")
		lA := landscapeFixture("dev", "kden-p-project-a", "Dev A")
		lB := landscapeFixture("dev", "kden-p-project-b", "Dev B")
		h := projectHandlerWith(project, lA, lB)

		ctx := ctxWithProjectRoles(auth.ProjectRoles{"project-a": {"admin"}})
		resp, err := h.ListLandscapesV1(ctx, openapi.ListLandscapesV1RequestObject{ProjectId: "project-a"})
		Expect(err).NotTo(HaveOccurred())

		ok := resp.(openapi.ListLandscapesV1200JSONResponse)
		Expect(ok.Data).To(HaveLen(1))
		Expect(ok.Data[0].Id).To(Equal("dev"))
	})
})

package project_test

import (
	"context"
	"errors"
	"testing"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/auth"
	"github.com/konfidence-project/konfidence/internal/project"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func projectScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := konfidence.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	return scheme
}

func TestRepositoryGet(t *testing.T) {
	want := &konfidence.Project{ObjectMeta: metav1.ObjectMeta{Name: "my-project"}}
	k8s := fake.NewClientBuilder().WithScheme(projectScheme(t)).WithObjects(want).Build()

	got, err := project.NewRepository(k8s).Get(context.Background(), want.Name, auth.ProjectRoles{"my-project": {"admin"}})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != want.Name {
		t.Fatalf("expected project %q, got %q", want.Name, got.Name)
	}
}

func TestRepositoryGetNotFound(t *testing.T) {
	k8s := fake.NewClientBuilder().WithScheme(projectScheme(t)).Build()

	_, err := project.NewRepository(k8s).Get(context.Background(), "missing", auth.ProjectRoles{"missing": {"admin"}})

	if !errors.Is(err, project.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestRepositoryGetForbidden(t *testing.T) {
	want := &konfidence.Project{ObjectMeta: metav1.ObjectMeta{Name: "my-project"}}
	k8s := fake.NewClientBuilder().WithScheme(projectScheme(t)).WithObjects(want).Build()

	_, err := project.NewRepository(k8s).Get(context.Background(), "my-project", auth.ProjectRoles{})

	if !errors.Is(err, project.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestRepositoryList(t *testing.T) {
	k8s := fake.NewClientBuilder().WithScheme(projectScheme(t)).WithObjects(
		&konfidence.Project{ObjectMeta: metav1.ObjectMeta{Name: "one"}},
		&konfidence.Project{ObjectMeta: metav1.ObjectMeta{Name: "two"}},
	).Build()

	projects, err := project.NewRepository(k8s).List(context.Background(), auth.ProjectRoles{
		"one": {"admin"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 1 || projects[0].Name != "one" {
		t.Fatalf("expected only authorized project, got %#v", projects)
	}
}

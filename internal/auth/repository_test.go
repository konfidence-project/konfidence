package auth_test

import (
	"context"
	"testing"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/auth"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestRepositoryGetProjectRoles(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := konfidence.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	k8s := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&konfidence.Project{
			ObjectMeta: metav1.ObjectMeta{Name: "accessible"},
			Spec: konfidence.ProjectSpec{RoleBindings: map[string]konfidence.Subjects{
				"viewer": {{Session: &konfidence.SessionSubject{MemberOf: []string{"all-users"}}}},
				"admin":  {{Session: &konfidence.SessionSubject{MemberOf: []string{"platform-engineers"}}}},
				"ci":     {{JWKS: &konfidence.JWKSSubject{}}},
			}},
		},
		&konfidence.Project{
			ObjectMeta: metav1.ObjectMeta{Name: "hidden"},
			Spec: konfidence.ProjectSpec{RoleBindings: map[string]konfidence.Subjects{
				"admin": {{Session: &konfidence.SessionSubject{MemberOf: []string{"platform-managers"}}}},
			}},
		},
	).Build()

	roles, err := auth.NewRepository(k8s).GetProjectRoles(
		context.Background(), []string{"all-users", "platform-engineers"},
	)
	if err != nil {
		t.Fatal(err)
	}
	accessible := roles["accessible"]
	if len(accessible) != 2 || accessible[0] != "admin" || accessible[1] != "viewer" {
		t.Fatalf("unexpected roles: %v", accessible)
	}
	if _, ok := roles["hidden"]; ok {
		t.Fatal("unexpected access to hidden project")
	}
}

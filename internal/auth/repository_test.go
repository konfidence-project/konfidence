package auth_test

import (
	"context"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/auth"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Repository", func() {
	It("gets project roles for the user's groups", func() {
		scheme := runtime.NewScheme()
		Expect(konfidence.AddToScheme(scheme)).To(Succeed())
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

		roles, err := auth.NewRepository(k8s, 15*time.Minute).GetProjectRoles(
			context.Background(), []string{"all-users", "platform-engineers"},
		)
		Expect(err).NotTo(HaveOccurred())
		Expect(roles["accessible"]).To(Equal([]string{"admin", "viewer"}))
		Expect(roles).NotTo(HaveKey("hidden"))
	})
})

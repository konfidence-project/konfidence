package auth

import (
	"context"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type stubTokenVerifier struct {
	results map[verifierKey]*verifiedToken
	errors  map[verifierKey]error
	calls   map[verifierKey]int
}

func (v *stubTokenVerifier) Verify(
	_ context.Context,
	_ string,
	endpoint string,
	audience string,
) (*verifiedToken, error) {
	key := verifierKey{endpoint: endpoint, audience: audience}
	v.calls[key]++
	if err := v.errors[key]; err != nil {
		return nil, err
	}
	return v.results[key], nil
}

func newTokenTestRepository(verifier tokenVerifier, projects ...*konfidence.Project) *k8sRepository {
	GinkgoHelper()
	scheme := runtime.NewScheme()
	Expect(konfidence.AddToScheme(scheme)).To(Succeed())

	objects := make([]runtime.Object, 0, len(projects))
	for _, project := range projects {
		objects = append(objects, project)
	}

	return &k8sRepository{
		reader: fake.NewClientBuilder().
			WithScheme(scheme).
			WithRuntimeObjects(objects...).
			Build(),
		tokenVerifier: verifier,
	}
}

var _ = Describe("Token identity", func() {
	It("maps matching token bindings to project roles", func() {
		const (
			endpoint = "https://issuer.example/.well-known/openid-configuration"
			audience = "konfidence"
		)
		key := verifierKey{endpoint: endpoint, audience: audience}
		verifier := &stubTokenVerifier{
			results: map[verifierKey]*verifiedToken{
				key: {
					subject: "workload-subject",
					claims: map[string]any{
						"sub":        "repo:konfidence-project/konfidence:ref:main",
						"repository": "konfidence-project/konfidence",
					},
				},
			},
			errors: make(map[verifierKey]error),
			calls:  make(map[verifierKey]int),
		}
		jwks := func(claims map[string]konfidence.GlobMatch) *konfidence.JWKSSubject {
			return &konfidence.JWKSSubject{Endpoint: endpoint, Audience: audience, Claims: claims}
		}
		repository := newTokenTestRepository(verifier, &konfidence.Project{
			ObjectMeta: metav1.ObjectMeta{Name: "project-a"},
			Spec: konfidence.ProjectSpec{RoleBindings: map[string]konfidence.Subjects{
				"viewer": {{JWKS: jwks(map[string]konfidence.GlobMatch{
					"repository": "konfidence-project/*",
				})}},
				"admin": {{JWKS: jwks(map[string]konfidence.GlobMatch{
					"sub":        "repo:konfidence-project/konfidence:*",
					"repository": "konfidence-project/konfidence",
				})}},
				"mismatch": {{JWKS: jwks(map[string]konfidence.GlobMatch{
					"repository": "different/*",
				})}},
				"session-only": {{Session: &konfidence.SessionSubject{MemberOf: []string{"admins"}}}},
			}},
		})

		identity, err := repository.AuthenticateToken(context.Background(), "signed-token")
		Expect(err).NotTo(HaveOccurred())
		Expect(identity.Subject).To(Equal("workload-subject"))
		Expect(identity.ProjectRoles).To(HaveKeyWithValue("project-a", []string{"admin", "viewer"}))
		Expect(verifier.calls[key]).To(Equal(1), "duplicate candidates should be verified once")
	})

	It("rejects an invalid token", func() {
		key := verifierKey{
			endpoint: "https://issuer.example/.well-known/openid-configuration",
			audience: "konfidence",
		}
		verifier := &stubTokenVerifier{
			results: make(map[verifierKey]*verifiedToken),
			errors:  map[verifierKey]error{key: ErrInvalidBearerToken},
			calls:   make(map[verifierKey]int),
		}
		repository := newTokenTestRepository(verifier, &konfidence.Project{
			ObjectMeta: metav1.ObjectMeta{Name: "project-a"},
			Spec: konfidence.ProjectSpec{RoleBindings: map[string]konfidence.Subjects{
				"admin": {{JWKS: &konfidence.JWKSSubject{
					Endpoint: key.endpoint,
					Audience: key.audience,
					Claims:   map[string]konfidence.GlobMatch{"sub": "repo:*"},
				}}},
			}},
		})

		identity, err := repository.AuthenticateToken(context.Background(), "invalid-token")
		Expect(identity).To(BeNil())
		Expect(err).To(MatchError(ErrInvalidBearerToken))
	})

	DescribeTable("matches glob patterns",
		func(pattern, value string, want bool) {
			Expect(matchesGlob(pattern, value)).To(Equal(want))
		},
		Entry("exactly", "repo:owner/name", "repo:owner/name", true),
		Entry("with a wildcard", "repo:owner/*", "repo:owner/name:ref:main", true),
		Entry("when the wildcard is empty", "repo:*", "repo:", true),
		Entry("only from the start", "owner/*", "repo:owner/name", false),
		Entry("only to the end", "repo:owner", "repo:owner/name", false),
		Entry("with regex metacharacters treated literally", "repo:owner/name.with+meta", "repo:owner/nameXwithmeta", false),
	)
})

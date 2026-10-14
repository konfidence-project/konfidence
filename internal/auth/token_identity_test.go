package auth

import (
	"context"
	"fmt"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type stubTokenVerifier struct {
	results            map[string]*verifiedToken
	errors             map[string]error
	calls              map[string]int
	requestedAudiences map[string][]string
}

func newStubTokenVerifier() *stubTokenVerifier {
	return &stubTokenVerifier{
		results:            make(map[string]*verifiedToken),
		errors:             make(map[string]error),
		calls:              make(map[string]int),
		requestedAudiences: make(map[string][]string),
	}
}

func (v *stubTokenVerifier) Verify(_ context.Context, _ string, endpoint string, allowedAudiences []string) (*verifiedToken, error) {
	v.calls[endpoint]++
	v.requestedAudiences[endpoint] = append([]string(nil), allowedAudiences...)
	if err := v.errors[endpoint]; err != nil {
		return nil, err
	}

	result, ok := v.results[endpoint]
	if !ok || result == nil {
		return nil, fmt.Errorf("unexpected token verification for endpoint %q", endpoint)
	}

	return result, nil
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

		verifier := newStubTokenVerifier()
		verifier.results[endpoint] = &verifiedToken{
			subject:   "workload-subject",
			audiences: []string{audience},
			claims: map[string]any{
				"sub":        "repo:konfidence-project/konfidence:ref:main",
				"repository": "konfidence-project/konfidence",
			},
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
		Expect(verifier.calls[endpoint]).To(Equal(1))
		Expect(verifier.requestedAudiences[endpoint]).To(ConsistOf(audience))
	})

	It("rejects an invalid token", func() {
		const (
			endpoint = "https://issuer.example/.well-known/openid-configuration"
			audience = "konfidence"
		)

		verifier := newStubTokenVerifier()
		verifier.errors[endpoint] = ErrInvalidBearerToken
		repository := newTokenTestRepository(verifier, &konfidence.Project{
			ObjectMeta: metav1.ObjectMeta{Name: "project-a"},
			Spec: konfidence.ProjectSpec{RoleBindings: map[string]konfidence.Subjects{
				"admin": {{
					JWKS: &konfidence.JWKSSubject{
						Endpoint: endpoint,
						Audience: audience,
						Claims: map[string]konfidence.GlobMatch{
							"sub": "repo:*",
						},
					},
				}},
			}},
		})

		identity, err := repository.AuthenticateToken(context.Background(), "invalid-token")

		Expect(identity).To(BeNil())
		Expect(err).To(MatchError(ErrInvalidBearerToken))
		Expect(verifier.calls[endpoint]).To(Equal(1))
		Expect(verifier.requestedAudiences[endpoint]).To(
			ConsistOf(audience),
		)
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

	It("verifies once per endpoint and applies each verified audience", func() {
		const endpoint = "https://issuer.example/.well-known/openid-configuration"
		verifier := newStubTokenVerifier()
		verifier.results[endpoint] = &verifiedToken{
			subject: "workload-subject",
			audiences: []string{
				"audience-a",
				"audience-b",
			},
			claims: map[string]any{
				"repository": "konfidence-project/konfidence",
			},
		}

		repository := newTokenTestRepository(
			verifier,
			&konfidence.Project{
				ObjectMeta: metav1.ObjectMeta{Name: "project-a"},
				Spec: konfidence.ProjectSpec{
					RoleBindings: map[string]konfidence.Subjects{
						"viewer": {{
							JWKS: &konfidence.JWKSSubject{
								Endpoint: endpoint,
								Audience: "audience-a",
								Claims: map[string]konfidence.GlobMatch{
									"repository": "konfidence-project/*",
								},
							},
						}},
						"admin": {{
							JWKS: &konfidence.JWKSSubject{
								Endpoint: endpoint,
								Audience: "audience-b",
								Claims: map[string]konfidence.GlobMatch{
									"repository": "konfidence-project/*",
								},
							},
						}},
					},
				},
			},
		)

		identity, err := repository.AuthenticateToken(context.Background(), "signed-token")

		Expect(err).NotTo(HaveOccurred())
		Expect(identity.ProjectRoles).To(HaveKeyWithValue("project-a", []string{"admin", "viewer"}))
		Expect(verifier.calls[endpoint]).To(Equal(1))
		Expect(verifier.requestedAudiences[endpoint]).To(ConsistOf("audience-a", "audience-b"))
	})
})

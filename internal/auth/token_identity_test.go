package auth

import (
	"context"
	"errors"
	"testing"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
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
	key := verifierKey{
		endpoint: endpoint,
		audience: audience,
	}
	v.calls[key]++

	if err := v.errors[key]; err != nil {
		return nil, err
	}
	return v.results[key], nil
}

func newTokenTestRepository(
	t *testing.T,
	verifier tokenVerifier,
	projects ...*konfidence.Project,
) *k8sRepository {
	t.Helper()

	scheme := runtime.NewScheme()
	if err := konfidence.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

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

func TestAuthenticateTokenMapsProjectRoles(t *testing.T) {
	const (
		endpoint = "https://issuer.example/.well-known/openid-configuration"
		audience = "konfidence"
	)

	key := verifierKey{
		endpoint: endpoint,
		audience: audience,
	}
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

	jwks := func(
		claims map[string]konfidence.GlobMatch,
	) *konfidence.JWKSSubject {
		return &konfidence.JWKSSubject{
			Endpoint: endpoint,
			Audience: audience,
			Claims:   claims,
		}
	}

	repository := newTokenTestRepository(
		t,
		verifier,
		&konfidence.Project{
			ObjectMeta: metav1.ObjectMeta{
				Name: "project-a",
			},
			Spec: konfidence.ProjectSpec{
				RoleBindings: map[string]konfidence.Subjects{
					"viewer": {{
						JWKS: jwks(map[string]konfidence.GlobMatch{
							"repository": "konfidence-project/*",
						}),
					}},
					"admin": {{
						JWKS: jwks(map[string]konfidence.GlobMatch{
							"sub":        "repo:konfidence-project/konfidence:*",
							"repository": "konfidence-project/konfidence",
						}),
					}},
					"mismatch": {{
						JWKS: jwks(map[string]konfidence.GlobMatch{
							"repository": "different/*",
						}),
					}},
					"session-only": {{
						Session: &konfidence.SessionSubject{
							MemberOf: []string{"admins"},
						},
					}},
				},
			},
		},
	)

	identity, err := repository.AuthenticateToken(
		context.Background(),
		"signed-token",
	)
	if err != nil {
		t.Fatal(err)
	}

	if identity.Subject != "workload-subject" {
		t.Fatalf("unexpected subject: %q", identity.Subject)
	}

	roles := identity.ProjectRoles["project-a"]
	if len(roles) != 2 ||
		roles[0] != "admin" ||
		roles[1] != "viewer" {
		t.Fatalf("unexpected roles: %v", roles)
	}

	if verifier.calls[key] != 1 {
		t.Fatalf(
			"expected duplicate candidates to be verified once, got %d",
			verifier.calls[key],
		)
	}
}

func TestAuthenticateTokenRejectsInvalidToken(t *testing.T) {
	key := verifierKey{
		endpoint: "https://issuer.example/.well-known/openid-configuration",
		audience: "konfidence",
	}
	verifier := &stubTokenVerifier{
		results: make(map[verifierKey]*verifiedToken),
		errors: map[verifierKey]error{
			key: ErrInvalidBearerToken,
		},
		calls: make(map[verifierKey]int),
	}

	repository := newTokenTestRepository(
		t,
		verifier,
		&konfidence.Project{
			ObjectMeta: metav1.ObjectMeta{
				Name: "project-a",
			},
			Spec: konfidence.ProjectSpec{
				RoleBindings: map[string]konfidence.Subjects{
					"admin": {{
						JWKS: &konfidence.JWKSSubject{
							Endpoint: key.endpoint,
							Audience: key.audience,
							Claims: map[string]konfidence.GlobMatch{
								"sub": "repo:*",
							},
						},
					}},
				},
			},
		},
	)

	identity, err := repository.AuthenticateToken(
		context.Background(),
		"invalid-token",
	)

	if identity != nil {
		t.Fatalf("expected nil identity, got %+v", identity)
	}
	if !errors.Is(err, ErrInvalidBearerToken) {
		t.Fatalf(
			"expected ErrInvalidBearerToken, got %v",
			err,
		)
	}
}

func TestMatchesGlob(t *testing.T) {
	tests := map[string]struct {
		pattern string
		value   string
		want    bool
	}{
		"exact": {
			pattern: "repo:owner/name",
			value:   "repo:owner/name",
			want:    true,
		},
		"wildcard": {
			pattern: "repo:owner/*",
			value:   "repo:owner/name:ref:main",
			want:    true,
		},
		"wildcard matches empty": {
			pattern: "repo:*",
			value:   "repo:",
			want:    true,
		},
		"anchored at start": {
			pattern: "owner/*",
			value:   "repo:owner/name",
			want:    false,
		},
		"anchored at end": {
			pattern: "repo:owner",
			value:   "repo:owner/name",
			want:    false,
		},
		"regex metacharacters are literal": {
			pattern: "repo:owner/name.with+meta",
			value:   "repo:owner/nameXwithmeta",
			want:    false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := matchesGlob(test.pattern, test.value)
			if got != test.want {
				t.Fatalf(
					"matchesGlob(%q, %q)=%t, want %t",
					test.pattern,
					test.value,
					got,
					test.want,
				)
			}
		})
	}
}

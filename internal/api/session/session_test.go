package session

import (
	"context"
	"reflect"
	"time"

	"github.com/konfidence-project/konfidence/internal/api/oidc"
	"github.com/konfidence-project/konfidence/internal/auth"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
)

type foreignContextKey string

var _ = ginkgo.Describe("Session context", func() {
	ginkgo.It("stores and retrieves the session context", func() {
		session := &Session{
			Context: Context{
				ID:                "session-id",
				Subject:           "subject-id",
				Name:              new("Test User"),
				Email:             new("test@example.com"),
				GivenName:         new("Test"),
				FamilyName:        new("User"),
				PreferredUsername: new("test.user"),
				ProjectRoles: auth.ProjectRoles{
					"project-a": {"admin", "viewer"},
				},
			},
			Groups:      []string{"developers"},
			AccessToken: "access-token",
		}

		ctx := NewContext(context.Background(), session)

		stored, err := FromContext(ctx)

		Expect(err).NotTo(HaveOccurred())
		Expect(stored).To(Equal(&session.Context))
	})

	ginkgo.It("stores only the context-safe session fields", func() {
		session := &Session{
			Context: Context{
				ID:      "session-id",
				Subject: "subject-id",
				Email:   new("test@example.com"),
			},
			Groups:       []string{"admins"},
			AccessToken:  "secret-access-token",
			RefreshToken: new("secret-refresh-token"),
		}

		ctx := NewContext(context.Background(), session)
		stored, err := FromContext(ctx)

		Expect(err).NotTo(HaveOccurred())
		Expect(stored.ID).To(Equal("session-id"))
		Expect(stored.Email).NotTo(BeNil())
		Expect(*stored.Email).To(Equal("test@example.com"))
	})

	ginkgo.It("copies the context value when creating the context", func() {
		session := &Session{
			Context: Context{
				ID:   "original-id",
				Name: new("Original Name"),
			},
		}

		ctx := NewContext(context.Background(), session)

		session.ID = "changed-id"
		session.Name = new("Changed Name")

		stored, err := FromContext(ctx)

		Expect(err).NotTo(HaveOccurred())
		Expect(stored.ID).To(Equal("original-id"))
		Expect(stored.Name).NotTo(BeNil())
		Expect(*stored.Name).To(Equal("Original Name"))
	})

	ginkgo.It("returns an error when no session exists", func() {
		stored, err := FromContext(context.Background())

		Expect(stored).To(BeNil())
		Expect(err).To(MatchError("session not found in context"))
	})

	ginkgo.It("does not read values stored under a different key type", func() {
		ctx := context.WithValue(
			context.Background(),
			foreignContextKey("session"),
			Context{ID: "session-id"},
		)

		stored, err := FromContext(ctx)

		Expect(stored).To(BeNil())
		Expect(err).To(MatchError("session not found in context"))
	})

	ginkgo.It("stores a workload request identity", func() {
		identity := Context{
			Subject: "workload-subject",
			ProjectRoles: auth.ProjectRoles{
				"project-a": {"admin"},
			},
		}

		ctx := NewRequestContext(context.Background(), identity)
		identity.Subject = "changed-subject"
		stored, err := FromContext(ctx)

		Expect(err).NotTo(HaveOccurred())
		Expect(stored).To(Equal(&Context{
			Subject: "workload-subject",
			ProjectRoles: auth.ProjectRoles{
				"project-a": {"admin"},
			},
		}))
	})

	ginkgo.It("isolates project roles from mutations", func() {
		roles := auth.ProjectRoles{
			"project-a": {"admin", "viewer"},
		}
		session := &Session{
			Context: Context{
				ProjectRoles: roles,
			},
		}

		ctx := NewContext(context.Background(), session)

		// mutating the source must not affect the context.
		const changed = "changed"
		roles["project-a"][0] = changed
		roles["project-b"] = []string{"admin"}

		stored, err := FromContext(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(stored.ProjectRoles).To(Equal(auth.ProjectRoles{
			"project-a": {"admin", "viewer"},
		}))

		// mutating a retrieved value must not affect subsequent retrievals.
		stored.ProjectRoles["project-a"][0] = changed
		delete(stored.ProjectRoles, "project-a")

		storedAgain, err := FromContext(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(storedAgain.ProjectRoles).To(Equal(auth.ProjectRoles{
			"project-a": {"admin", "viewer"},
		}))
	})
})

var _ = ginkgo.Describe("Session token expiry", func() {
	ginkgo.DescribeTable(
		"identifies zero expiry",
		func(tokenExpiry int64, expected bool) {
			sess := Session{TokenExpiry: tokenExpiry}

			Expect(sess.IsTokenExpiryZero()).To(Equal(expected))
		},
		ginkgo.Entry("zero", int64(0), true),
		ginkgo.Entry("future Unix timestamp", int64(1_800_000_000), false),
		ginkgo.Entry("negative value", int64(-1), false),
	)

	ginkgo.It("converts a zero time to zero", func() {
		Expect(UnixExpiry(time.Time{})).To(BeZero())
	})

	ginkgo.It("converts a non-zero time to its Unix timestamp", func() {
		tokenExpiry := time.Date(2030, time.January, 1, 0, 0, 0, 0, time.UTC)

		Expect(UnixExpiry(tokenExpiry)).To(Equal(tokenExpiry.Unix()))
	})
})

var _ = ginkgo.Describe("OIDC session data", func() {
	const refreshToken = "refresh-token"

	ginkgo.It("applies OIDC data while preserving session context", func() {
		expiry := time.Date(2030, time.January, 1, 0, 0, 0, 0, time.UTC)

		claims := oidc.IDTokenAdditionalClaims{
			Groups:            []string{"developers", "admins"},
			Name:              new("Test User"),
			Email:             new("test@example.com"),
			GivenName:         new("Test"),
			FamilyName:        new("User"),
			PreferredUsername: new("test.user"),
		}
		projectRoles := auth.ProjectRoles{
			"project-a": {"admin"},
		}
		sess := Session{
			Context: Context{
				ID:           "session-id",
				ProjectRoles: projectRoles,
			},
		}

		sess.ApplyOIDCValues("subject-id", claims, "access-token", lo.ToPtr(refreshToken), expiry)
		Expect(sess).To(Equal(Session{
			Context: Context{
				ID:                "session-id",
				Subject:           "subject-id",
				Name:              claims.Name,
				Email:             claims.Email,
				GivenName:         claims.GivenName,
				FamilyName:        claims.FamilyName,
				PreferredUsername: claims.PreferredUsername,
				ProjectRoles:      projectRoles,
			},
			Groups:       []string{"developers", "admins"},
			AccessToken:  "access-token",
			RefreshToken: lo.ToPtr(refreshToken),
			TokenExpiry:  expiry.Unix(),
		}))

		claims.Groups[0] = "changed"
		Expect(sess.Groups).To(Equal([]string{"developers", "admins"}))
	})
})

var _ = ginkgo.Describe("Session cloning", func() {
	ginkgo.It("documents every field handled by cloneSession", func() {
		expectFields[Session](map[string]reflect.Type{
			"Context":      reflect.TypeFor[Context](),
			"Groups":       reflect.TypeFor[[]string](),
			"AccessToken":  reflect.TypeFor[string](),
			"RefreshToken": reflect.TypeFor[*string](),
			"TokenExpiry":  reflect.TypeFor[int64](),
		})

		expectFields[Context](map[string]reflect.Type{
			"ID":                reflect.TypeFor[string](),
			"Subject":           reflect.TypeFor[string](),
			"Name":              reflect.TypeFor[*string](),
			"Email":             reflect.TypeFor[*string](),
			"GivenName":         reflect.TypeFor[*string](),
			"FamilyName":        reflect.TypeFor[*string](),
			"PreferredUsername": reflect.TypeFor[*string](),
			"ProjectRoles":      reflect.TypeFor[auth.ProjectRoles](),
		})
	})
})

func expectFields[T any](expected map[string]reflect.Type) {
	ginkgo.GinkgoHelper()
	valueType := reflect.TypeFor[T]()
	for field := range valueType.Fields() {
		expectedType, exists := expected[field.Name]

		Expect(exists).To(BeTrue(),
			"%s contains a new or renamed field %q; review cloneSession method", valueType, field.Name)
		if !exists {
			continue
		}

		Expect(field.Type).To(Equal(expectedType), "%s.%s changed type; review cloneSession method", valueType, field.Name)
	}

	Expect(valueType.NumField()).To(Equal(len(expected)), "%s field count changed; review cloneSession method", valueType)
}

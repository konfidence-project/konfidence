package session

import (
	"context"
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
				Name:              new("Test User"),
				Email:             new("test@example.com"),
				GivenName:         new("Test"),
				FamilyName:        new("User"),
				PreferredUsername: new("test.user"),
				ProjectRoles: auth.ProjectRoles{
					"project-a": {"admin", "viewer"},
				},
			},
			Subject:     "subject-id",
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
				ID:    "session-id",
				Email: new("test@example.com"),
			},
			Subject:      "subject-id",
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
				Name:              claims.Name,
				Email:             claims.Email,
				GivenName:         claims.GivenName,
				FamilyName:        claims.FamilyName,
				PreferredUsername: claims.PreferredUsername,
				ProjectRoles:      projectRoles,
			},
			Subject:      "subject-id",
			Groups:       []string{"developers", "admins"},
			AccessToken:  "access-token",
			RefreshToken: lo.ToPtr(refreshToken),
			TokenExpiry:  expiry.Unix(),
		}))

		claims.Groups[0] = "changed"
		Expect(sess.Groups).To(Equal([]string{"developers", "admins"}))
	})
})

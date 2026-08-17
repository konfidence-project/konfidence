package session

import (
	"context"

	"github.com/konfidence-project/konfidence/internal/auth"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

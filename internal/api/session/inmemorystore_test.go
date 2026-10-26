package session

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/konfidence-project/konfidence/internal/api/config"
	"github.com/konfidence-project/konfidence/internal/auth"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = ginkgo.Describe("InMemoryStore", func() {
	var (
		ctx   context.Context
		store *InMemoryStore
	)

	ginkgo.BeforeEach(func() {
		ctx = context.Background()
		store = NewInMemoryStore(config.Parsed{
			Session: config.ParsedSessionConfig{
				Expiration: time.Minute,
			},
		})
	})

	ginkgo.It("rejects a nil session", func() {
		id, err := store.Save(ctx, nil)

		Expect(id).To(BeEmpty())
		Expect(err).To(MatchError(
			"failed to store session: session is empty",
		))
	})

	ginkgo.It("saves and retrieves a session", func() {
		input := &Session{
			Groups:       []string{"developers", "admins"},
			AccessToken:  "access-token",
			RefreshToken: new("refresh-token"),
			TokenExpiry:  time.Now().Add(time.Hour).Unix(),
			Context: Context{
				Subject:           "subject-id",
				Name:              new("Test User"),
				Email:             new("test@example.com"),
				GivenName:         new("Test"),
				FamilyName:        new("User"),
				PreferredUsername: new("test.user"),
			},
		}

		id, err := store.Save(ctx, input)

		Expect(err).NotTo(HaveOccurred())
		Expect(id).NotTo(BeEmpty())
		Expect(input.ID).To(Equal(id))

		parsedID, err := uuid.Parse(id)
		Expect(err).NotTo(HaveOccurred())
		Expect(parsedID.Version()).To(Equal(uuid.Version(7)))

		stored, err := store.Get(ctx, id)

		Expect(err).NotTo(HaveOccurred())
		Expect(stored).To(Equal(input))
	})

	ginkgo.It("isolates stored sessions from caller mutations", func() {
		input := &Session{
			Context: Context{
				Subject:           "original-subject",
				Name:              new("Original Name"),
				Email:             new("original@example.com"),
				GivenName:         new("Original"),
				FamilyName:        new("User"),
				PreferredUsername: new("original.user"),
				ProjectRoles: auth.ProjectRoles{
					"project-a": {"admin", "viewer"},
				},
			},
			Groups:       []string{"developers", "admins"},
			AccessToken:  "original-access-token",
			RefreshToken: new("original-refresh-token"),
			TokenExpiry:  123456,
		}

		id, err := store.Save(ctx, input)
		Expect(err).NotTo(HaveOccurred())

		expected := &Session{
			Context: Context{
				ID:                id,
				Subject:           "original-subject",
				Name:              new("Original Name"),
				Email:             new("original@example.com"),
				GivenName:         new("Original"),
				FamilyName:        new("User"),
				PreferredUsername: new("original.user"),
				ProjectRoles: auth.ProjectRoles{
					"project-a": {"admin", "viewer"},
				},
			},
			Groups:       []string{"developers", "admins"},
			AccessToken:  "original-access-token",
			RefreshToken: new("original-refresh-token"),
			TokenExpiry:  123456,
		}

		// Mutating the Save input must not affect the cached session.
		mutateSession(input)

		stored, err := store.Get(ctx, id)
		Expect(err).NotTo(HaveOccurred())
		Expect(stored).To(Equal(expected))

		// Mutating a Get result must not affect subsequent retrievals.
		mutateSession(stored)

		storedAgain, err := store.Get(ctx, id)
		Expect(err).NotTo(HaveOccurred())
		Expect(storedAgain).To(Equal(expected))
	})

	ginkgo.It("returns nil for an unknown session", func() {
		stored, err := store.Get(ctx, "unknown-session")

		Expect(err).NotTo(HaveOccurred())
		Expect(stored).To(BeNil())
	})

	ginkgo.It("deletes a session", func() {
		id, err := store.Save(ctx, &Session{
			Context: Context{Subject: "subject-id"},
		})
		Expect(err).NotTo(HaveOccurred())

		Expect(store.Delete(ctx, id)).To(Succeed())

		stored, err := store.Get(ctx, id)
		Expect(err).NotTo(HaveOccurred())
		Expect(stored).To(BeNil())
	})

	ginkgo.It("allows deletion of an unknown session", func() {
		Expect(store.Delete(ctx, "unknown-session")).To(Succeed())
	})

	ginkgo.It("expires sessions after creation despite repeated access", func() {
		const expiration = 40 * time.Millisecond

		store = NewInMemoryStore(config.Parsed{
			Session: config.ParsedSessionConfig{
				Expiration: expiration,
			},
		})

		id, err := store.Save(ctx, &Session{
			Context: Context{Subject: "subject-id"},
		})
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() bool {
			stored, err := store.Get(ctx, id)
			Expect(err).NotTo(HaveOccurred())
			return stored == nil
		}, 3*expiration, expiration/10).Should(BeTrue())
	})
})

func mutateSession(session *Session) {
	session.ID = "changed-id"
	session.Subject = "changed-subject"
	*session.Name = "Changed Name"
	*session.Email = "changed@example.com"
	*session.GivenName = "Changed"
	*session.FamilyName = "Person"
	*session.PreferredUsername = "changed.user"

	session.ProjectRoles["project-a"][0] = "changed-role"
	session.ProjectRoles["project-b"] = []string{"admin"}

	session.Groups[0] = "changed-group"
	session.AccessToken = "changed-access-token"
	*session.RefreshToken = "changed-refresh-token"
	session.TokenExpiry = 654321
}

package session

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/konfidence-project/konfidence/internal/api/config"
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
			Subject:      "subject-id",
			Groups:       []string{"developers", "admins"},
			AccessToken:  "access-token",
			RefreshToken: new("refresh-token"),
			TokenExpiry:  time.Now().Add(time.Hour).Unix(),
			Context: Context{
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

	ginkgo.It("stores a value copy rather than the session pointer", func() {
		input := &Session{
			Subject:     "original-subject",
			AccessToken: "original-token",
		}

		id, err := store.Save(ctx, input)
		Expect(err).NotTo(HaveOccurred())

		input.Subject = "changed-subject"
		input.AccessToken = "changed-token"

		stored, err := store.Get(ctx, id)

		Expect(err).NotTo(HaveOccurred())
		Expect(stored.Subject).To(Equal("original-subject"))
		Expect(stored.AccessToken).To(Equal("original-token"))
	})

	ginkgo.It("returns nil for an unknown session", func() {
		stored, err := store.Get(ctx, "unknown-session")

		Expect(err).NotTo(HaveOccurred())
		Expect(stored).To(BeNil())
	})

	ginkgo.It("deletes a session", func() {
		id, err := store.Save(ctx, &Session{
			Subject: "subject-id",
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
			Subject: "subject-id",
		})
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() bool {
			stored, err := store.Get(ctx, id)
			Expect(err).NotTo(HaveOccurred())
			return stored == nil
		}, 3*expiration, expiration/10).Should(BeTrue())
	})

	ginkgo.It("updates a session without renewing its expiration", func() {
		store = NewInMemoryStore(config.Parsed{
			Session: config.ParsedSessionConfig{
				Expiration: time.Minute,
			},
		})

		input := &Session{
			Subject:     "subject-id",
			AccessToken: "old-access-token",
		}

		id, err := store.Save(ctx, input)
		Expect(err).NotTo(HaveOccurred())

		entryBefore, found := store.cache.GetEntryQuietly(id)
		Expect(found).To(BeTrue())

		updated := *input
		updated.AccessToken = "new-access-token"

		Expect(store.Update(ctx, &updated)).To(Succeed())

		stored, err := store.Get(ctx, id)
		Expect(err).NotTo(HaveOccurred())
		Expect(stored.AccessToken).To(Equal("new-access-token"))

		entryAfter, found := store.cache.GetEntryQuietly(id)
		Expect(found).To(BeTrue())
		Expect(entryAfter.ExpiresAt()).To(Equal(entryBefore.ExpiresAt()))
	})
})

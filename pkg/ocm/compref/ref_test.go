package compref

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Parse", func() {
	Context("semver versions", func() {
		It("parses standard semver references", func() {
			ref, err := Parse("ghcr.io/org/components//github.com/org/app:1.0.0")

			Expect(err).ToNot(HaveOccurred())
			Expect(ref.Component).To(Equal("github.com/org/app"))
			Expect(ref.Version).To(Equal("1.0.0"))
		})

		It("parses complex semver with prerelease and metadata", func() {
			ref, err := Parse("ghcr.io/org/components//github.com/org/app:2.0.0-rc.1+build.123")

			Expect(err).ToNot(HaveOccurred())
			Expect(ref.Component).To(Equal("github.com/org/app"))
			Expect(ref.Version).To(Equal("2.0.0-rc.1+build.123"))
		})
	})

	Context("OCI tag aliases", func() {
		It("parses branch aliases", func() {
			ref, err := Parse("ghcr.io/org/components//github.com/org/app:main")

			Expect(err).ToNot(HaveOccurred())
			Expect(ref.Component).To(Equal("github.com/org/app"))
			Expect(ref.Version).To(Equal("main"))
		})

		It("parses environment aliases", func() {
			ref, err := Parse("ghcr.io/org/components//github.com/org/app:production")

			Expect(err).ToNot(HaveOccurred())
			Expect(ref.Component).To(Equal("github.com/org/app"))
			Expect(ref.Version).To(Equal("production"))
		})

		It("parses stability aliases", func() {
			ref, err := Parse("ghcr.io/org/components//github.com/org/app:latest")

			Expect(err).ToNot(HaveOccurred())
			Expect(ref.Component).To(Equal("github.com/org/app"))
			Expect(ref.Version).To(Equal("latest"))
		})

		It("parses custom timestamp aliases", func() {
			ref, err := Parse("ghcr.io/org/components//github.com/org/app:nightly-2024-03-15")

			Expect(err).ToNot(HaveOccurred())
			Expect(ref.Component).To(Equal("github.com/org/app"))
			Expect(ref.Version).To(Equal("nightly-2024-03-15"))
		})
	})

	Context("optional version", func() {
		It("rejects references without version by default", func() {
			_, err := Parse("ghcr.io/org/components//github.com/org/app")

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ErrInvalidComponentReference))
		})

		It("accepts references without version when validation is skipped", func() {
			ref, err := Parse("ghcr.io/org/components//github.com/org/app", WithoutValidation())

			Expect(err).ToNot(HaveOccurred())
			Expect(ref.Component).To(Equal("github.com/org/app"))
			Expect(ref.Version).To(BeEmpty())
		})
	})

	Context("version validation modes", func() {
		//nolint:dupl // Parse tests intentionally mirror Validate tests - same modes at different layers
		Context("VersionValidationSemverOnly", func() {
			It("accepts semver versions", func() {
				ref, err := Parse("ghcr.io/org/components//github.com/org/app:1.0.0",
					WithVersionValidation(VersionValidationSemverOnly))

				Expect(err).ToNot(HaveOccurred())
				Expect(ref.Version).To(Equal("1.0.0"))
			})

			It("accepts semver with prerelease and metadata", func() {
				ref, err := Parse("ghcr.io/org/components//github.com/org/app:2.0.0-rc.1+build.123",
					WithVersionValidation(VersionValidationSemverOnly))

				Expect(err).ToNot(HaveOccurred())
				Expect(ref.Version).To(Equal("2.0.0-rc.1+build.123"))
			})

			It("accepts semver with v prefix", func() {
				ref, err := Parse("ghcr.io/org/components//github.com/org/app:v3.2.1",
					WithVersionValidation(VersionValidationSemverOnly))

				Expect(err).ToNot(HaveOccurred())
				Expect(ref.Version).To(Equal("v3.2.1"))
			})

			It("rejects alias versions like 'latest'", func() {
				_, err := Parse("ghcr.io/org/components//github.com/org/app:latest",
					WithVersionValidation(VersionValidationSemverOnly))

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("not a valid semantic version")))
				Expect(errors.Is(err, ErrInvalidComponentReference)).To(BeTrue())
			})

			It("rejects alias versions like 'main'", func() {
				_, err := Parse("ghcr.io/org/components//github.com/org/app:main",
					WithVersionValidation(VersionValidationSemverOnly))

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("not a valid semantic version")))
				Expect(errors.Is(err, ErrInvalidComponentReference)).To(BeTrue())
			})

			It("rejects custom aliases", func() {
				_, err := Parse("ghcr.io/org/components//github.com/org/app:production",
					WithVersionValidation(VersionValidationSemverOnly))

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("not a valid semantic version")))
				Expect(errors.Is(err, ErrInvalidComponentReference)).To(BeTrue())
			})
		})

		//nolint:dupl // Parse tests intentionally mirror Validate tests - same modes at different layers
		Context("VersionValidationAliasOnly", func() {
			It("accepts alias versions like 'latest'", func() {
				ref, err := Parse("ghcr.io/org/components//github.com/org/app:latest",
					WithVersionValidation(VersionValidationAliasOnly))

				Expect(err).ToNot(HaveOccurred())
				Expect(ref.Version).To(Equal("latest"))
			})

			It("accepts alias versions like 'main'", func() {
				ref, err := Parse("ghcr.io/org/components//github.com/org/app:main",
					WithVersionValidation(VersionValidationAliasOnly))

				Expect(err).ToNot(HaveOccurred())
				Expect(ref.Version).To(Equal("main"))
			})

			It("accepts custom aliases", func() {
				ref, err := Parse("ghcr.io/org/components//github.com/org/app:production",
					WithVersionValidation(VersionValidationAliasOnly))

				Expect(err).ToNot(HaveOccurred())
				Expect(ref.Version).To(Equal("production"))
			})

			It("rejects semver versions", func() {
				_, err := Parse("ghcr.io/org/components//github.com/org/app:1.0.0",
					WithVersionValidation(VersionValidationAliasOnly))

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("is a semantic version but only aliases are allowed")))
				Expect(errors.Is(err, ErrInvalidComponentReference)).To(BeTrue())
			})

			It("rejects semver with prerelease", func() {
				_, err := Parse("ghcr.io/org/components//github.com/org/app:2.0.0-rc.1",
					WithVersionValidation(VersionValidationAliasOnly))

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("is a semantic version but only aliases are allowed")))
				Expect(errors.Is(err, ErrInvalidComponentReference)).To(BeTrue())
			})

			It("rejects semver with v prefix", func() {
				_, err := Parse("ghcr.io/org/components//github.com/org/app:v3.2.1",
					WithVersionValidation(VersionValidationAliasOnly))

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("is a semantic version but only aliases are allowed")))
				Expect(errors.Is(err, ErrInvalidComponentReference)).To(BeTrue())
			})
		})

		Context("VersionValidationNoVersion", func() {
			It("accepts a bare component reference without a version", func() {
				ref, err := Parse("ghcr.io/org/components//github.com/org/app",
					WithVersionValidation(VersionValidationNoVersion))

				Expect(err).ToNot(HaveOccurred())
				Expect(ref.Version).To(BeEmpty())
			})

			It("rejects a semver version", func() {
				_, err := Parse("ghcr.io/org/components//github.com/org/app:1.0.0",
					WithVersionValidation(VersionValidationNoVersion))

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("must not carry a version")))
				Expect(errors.Is(err, ErrInvalidComponentReference)).To(BeTrue())
			})

			It("rejects an alias version", func() {
				_, err := Parse("ghcr.io/org/components//github.com/org/app:latest",
					WithVersionValidation(VersionValidationNoVersion))

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("must not carry a version")))
				Expect(errors.Is(err, ErrInvalidComponentReference)).To(BeTrue())
			})
		})

		Context("VersionValidationPermissive (default)", func() {
			It("accepts semver versions", func() {
				ref, err := Parse("ghcr.io/org/components//github.com/org/app:1.0.0")

				Expect(err).ToNot(HaveOccurred())
				Expect(ref.Version).To(Equal("1.0.0"))
			})

			It("accepts alias versions", func() {
				ref, err := Parse("ghcr.io/org/components//github.com/org/app:latest")

				Expect(err).ToNot(HaveOccurred())
				Expect(ref.Version).To(Equal("latest"))
			})
		})
	})

	Context("malformed references", func() {
		It("returns error for missing component separator", func() {
			_, err := Parse("ghcr.io/org/components/github.com/org/app:1.0.0")

			Expect(err).To(HaveOccurred())
		})

		It("returns error for malformed repository path", func() {
			_, err := Parse("not-a-valid-reference")

			Expect(err).To(HaveOccurred())
		})

		It("returns error for empty string", func() {
			_, err := Parse("")

			Expect(err).To(HaveOccurred())
		})
	})
})

var _ = Describe("Validate", func() {
	Context("valid references", func() {
		It("accepts references with semver version", func() {
			ref, _ := Parse("ghcr.io/org/components//github.com/org/app:1.0.0")
			err := Validate(*ref)

			Expect(err).ToNot(HaveOccurred())
		})

		It("accepts references with alias version", func() {
			ref, _ := Parse("ghcr.io/org/components//github.com/org/app:latest")
			err := Validate(*ref)

			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("invalid references", func() {
		It("rejects references without version", func() {
			ref, _ := Parse("ghcr.io/org/components//github.com/org/app", WithoutValidation())
			err := Validate(*ref)

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ErrInvalidComponentReference))
		})
	})

	Context("version validation modes", func() {
		//nolint:dupl // Validate tests intentionally mirror Parse tests - same modes at different layers
		Context("VersionValidationSemverOnly", func() {
			It("accepts semver versions", func() {
				ref, _ := Parse("ghcr.io/org/components//github.com/org/app:1.0.0", WithoutValidation())
				err := Validate(*ref, WithVersionValidationMode(VersionValidationSemverOnly))

				Expect(err).ToNot(HaveOccurred())
			})

			It("rejects alias versions", func() {
				ref, _ := Parse("ghcr.io/org/components//github.com/org/app:latest", WithoutValidation())
				err := Validate(*ref, WithVersionValidationMode(VersionValidationSemverOnly))

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("not a valid semantic version")))
				Expect(errors.Is(err, ErrInvalidComponentReference)).To(BeTrue())
			})
		})

		//nolint:dupl // Validate tests intentionally mirror Parse tests - same modes at different layers
		Context("VersionValidationAliasOnly", func() {
			It("accepts alias versions", func() {
				ref, _ := Parse("ghcr.io/org/components//github.com/org/app:latest", WithoutValidation())
				err := Validate(*ref, WithVersionValidationMode(VersionValidationAliasOnly))

				Expect(err).ToNot(HaveOccurred())
			})

			It("rejects semver versions", func() {
				ref, _ := Parse("ghcr.io/org/components//github.com/org/app:1.0.0", WithoutValidation())
				err := Validate(*ref, WithVersionValidationMode(VersionValidationAliasOnly))

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("is a semantic version but only aliases are allowed")))
				Expect(errors.Is(err, ErrInvalidComponentReference)).To(BeTrue())
			})
		})

		Context("VersionValidationNoVersion", func() {
			It("accepts a reference without a version", func() {
				ref, _ := Parse("ghcr.io/org/components//github.com/org/app", WithoutValidation())
				err := Validate(*ref, WithVersionValidationMode(VersionValidationNoVersion))

				Expect(err).ToNot(HaveOccurred())
			})

			It("rejects a reference that carries a version", func() {
				ref, _ := Parse("ghcr.io/org/components//github.com/org/app:1.0.0", WithoutValidation())
				err := Validate(*ref, WithVersionValidationMode(VersionValidationNoVersion))

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("must not carry a version")))
				Expect(errors.Is(err, ErrInvalidComponentReference)).To(BeTrue())
			})
		})

		Context("VersionValidationPermissive (default)", func() {
			It("accepts semver versions", func() {
				ref, _ := Parse("ghcr.io/org/components//github.com/org/app:1.0.0", WithoutValidation())
				err := Validate(*ref)

				Expect(err).ToNot(HaveOccurred())
			})

			It("accepts alias versions", func() {
				ref, _ := Parse("ghcr.io/org/components//github.com/org/app:latest", WithoutValidation())
				err := Validate(*ref)

				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
})

package cmd_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/konfidence-project/konfidence/pkg/cmd"
)

var _ = Describe("Filter", func() {
	registered := map[string]func() error{
		"StageConfiguration": func() error { return nil },
		"VectorAssembly":     func() error { return nil },
		"VectorPromotion":    func() error { return nil },
	}

	all := func() map[string]bool {
		out := map[string]bool{}
		for n := range registered {
			out[n] = true
		}
		return out
	}

	Context("when the spec is empty or '*'", func() {
		It("returns the full set for an empty spec", func() {
			got, err := cmd.FilterEnabledControllers("", registered)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(all()))
		})

		It("returns the full set for '*'", func() {
			got, err := cmd.FilterEnabledControllers("*", registered)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(all()))
		})
	})

	Context("with literal tokens", func() {
		It("selects a single named controller", func() {
			got, err := cmd.FilterEnabledControllers("VectorAssembly", registered)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(map[string]bool{"VectorAssembly": true}))
		})

		It("selects multiple named controllers", func() {
			got, err := cmd.FilterEnabledControllers("VectorAssembly,VectorPromotion", registered)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(map[string]bool{
				"VectorAssembly":  true,
				"VectorPromotion": true,
			}))
		})

		It("tolerates whitespace around tokens", func() {
			got, err := cmd.FilterEnabledControllers(" VectorAssembly , VectorPromotion ", registered)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(map[string]bool{
				"VectorAssembly":  true,
				"VectorPromotion": true,
			}))
		})
	})

	Context("with negation", func() {
		It("excludes when negation precedes the wildcard", func() {
			got, err := cmd.FilterEnabledControllers("!VectorAssembly,*", registered)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(map[string]bool{
				"StageConfiguration": true,
				"VectorPromotion":    true,
			}))
		})

		It("excludes when negation follows the wildcard (order-independent)", func() {
			got, err := cmd.FilterEnabledControllers("*,!VectorAssembly", registered)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(map[string]bool{
				"StageConfiguration": true,
				"VectorPromotion":    true,
			}))
		})
	})

	Context("with wildcard tokens", func() {
		It("matches by prefix", func() {
			got, err := cmd.FilterEnabledControllers("Vector*", registered)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(map[string]bool{
				"VectorPromotion": true,
				"VectorAssembly":  true,
			}))
		})
	})

	Context("error cases", func() {
		It("errors when a literal token matches no registered controller", func() {
			_, err := cmd.FilterEnabledControllers("Typo", registered)
			Expect(err).To(MatchError(ContainSubstring("matches no registered controller")))
		})

		It("returns an error when a wildcard matches no registered controller", func() {
			_, err := cmd.FilterEnabledControllers("Nonexistent*", registered)
			Expect(err).To(MatchError(ContainSubstring("matches no registered controller")))
		})

		It("errors even when a typo is mixed with valid tokens", func() {
			_, err := cmd.FilterEnabledControllers("VectorAssembly,Typo", registered)
			Expect(err).To(MatchError(ContainSubstring("matches no registered controller")))
		})

		It("errors on a malformed glob", func() {
			_, err := cmd.FilterEnabledControllers("[", registered)
			Expect(err).To(MatchError(ContainSubstring("invalid controller filter glob")))
		})

		It("errors on a bare bang", func() {
			_, err := cmd.FilterEnabledControllers("!", registered)
			Expect(err).To(MatchError(ContainSubstring("bare")))
		})
	})
})

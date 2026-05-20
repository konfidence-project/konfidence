package cli_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/konfidence-project/konfidence/pkg/cli"
)

var _ = Describe("Filter", func() {
	registered := []string{"StageConfiguration", "VectorPromotion", "VectorAssembly"}

	all := func() map[string]bool {
		out := map[string]bool{}
		for _, n := range registered {
			out[n] = true
		}
		return out
	}

	Context("when the spec is empty or '*'", func() {
		It("returns the full set for an empty spec", func() {
			got, err := cli.Filter("", registered)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(all()))
		})

		It("returns the full set for '*'", func() {
			got, err := cli.Filter("*", registered)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(all()))
		})
	})

	Context("with literal tokens", func() {
		It("selects a single named controller", func() {
			got, err := cli.Filter("VectorAssembly", registered)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(map[string]bool{"VectorAssembly": true}))
		})

		It("selects multiple named controllers", func() {
			got, err := cli.Filter("VectorAssembly,VectorPromotion", registered)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(map[string]bool{
				"VectorAssembly":  true,
				"VectorPromotion": true,
			}))
		})

		It("tolerates whitespace around tokens", func() {
			got, err := cli.Filter(" VectorAssembly , VectorPromotion ", registered)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(map[string]bool{
				"VectorAssembly":  true,
				"VectorPromotion": true,
			}))
		})
	})

	Context("with negation", func() {
		It("excludes when negation precedes the wildcard", func() {
			got, err := cli.Filter("!VectorAssembly,*", registered)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(map[string]bool{
				"StageConfiguration": true,
				"VectorPromotion":    true,
			}))
		})

		It("excludes when negation follows the wildcard (order-independent)", func() {
			got, err := cli.Filter("*,!VectorAssembly", registered)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(map[string]bool{
				"StageConfiguration": true,
				"VectorPromotion":    true,
			}))
		})
	})

	Context("with wildcard tokens", func() {
		It("matches by prefix", func() {
			got, err := cli.Filter("Vector*", registered)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(Equal(map[string]bool{
				"VectorPromotion": true,
				"VectorAssembly":  true,
			}))
		})

		It("returns an empty set when a wildcard matches nothing (no error)", func() {
			got, err := cli.Filter("Nonexistent*", registered)
			Expect(err).NotTo(HaveOccurred())
			Expect(got).To(BeEmpty())
		})
	})

	Context("error cases", func() {
		It("errors when a literal token matches no registered controller", func() {
			_, err := cli.Filter("Typo", registered)
			Expect(err).To(MatchError(ContainSubstring("matches no registered controller")))
		})

		It("errors even when a typo is mixed with valid tokens", func() {
			_, err := cli.Filter("VectorAssembly,Typo", registered)
			Expect(err).To(MatchError(ContainSubstring("matches no registered controller")))
		})

		It("errors on a malformed glob", func() {
			_, err := cli.Filter("[", registered)
			Expect(err).To(MatchError(ContainSubstring("invalid controller filter glob")))
		})

		It("errors on a bare bang", func() {
			_, err := cli.Filter("!", registered)
			Expect(err).To(MatchError(ContainSubstring("bare")))
		})
	})
})

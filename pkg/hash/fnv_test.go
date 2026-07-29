package hash_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/konfidence-project/konfidence/pkg/hash"
)

var _ = Describe("FNV Hash Functions", func() {
	Describe("Fnv", func() {
		It("should return consistent results for the same input and length", func() {
			input := "test string"
			result1 := hash.Fnv(input, 8)
			result2 := hash.Fnv(input, 8)
			Expect(result1).To(Equal(result2))
		})

		It("should return different hashes for different inputs", func() {
			result1 := hash.Fnv("input1", 8)
			result2 := hash.Fnv("input2", 8)
			Expect(result1).NotTo(Equal(result2))
		})

		It("should return exactly the requested length", func() {
			for length := 1; length < 26; length++ {
				result := hash.Fnv("test input", length)
				Expect(result).To(HaveLen(length),
					"Expected exact length %d, got %d", length, len(result))
			}
		})

		It("should only use safe characters (no vowels, no 0 or 1)", func() {
			result := hash.Fnv("test input", 13)
			// SafeEncodeString uses: bcdfghjklmnpqrstvwxz2456789
			Expect(result).To(MatchRegexp("^[bcdfghjklmnpqrstvwxz2456789]+$"))
			Expect(result).NotTo(ContainSubstring("a"))
			Expect(result).NotTo(ContainSubstring("e"))
			Expect(result).NotTo(ContainSubstring("i"))
			Expect(result).NotTo(ContainSubstring("o"))
			Expect(result).NotTo(ContainSubstring("u"))
			Expect(result).NotTo(ContainSubstring("0"))
			Expect(result).NotTo(ContainSubstring("1"))
		})

		It("should handle empty string", func() {
			result := hash.Fnv("", 8)
			Expect(result).To(HaveLen(8))
			Expect(result).To(MatchRegexp("^[bcdfghjklmnpqrstvwxz2456789]+$"))
		})

		It("should handle special characters consistently", func() {
			specialInputs := []string{
				"special!@#$%^&*()",
				"newline\ncharacter",
				"tab\tcharacter",
				"null\x00byte",
				"",
			}

			for _, input := range specialInputs {
				result := hash.Fnv(input, 8)
				Expect(result).To(HaveLen(8))
				Expect(result).To(MatchRegexp("^[bcdfghjklmnpqrstvwxz2456789]+$"))
			}
		})

		It("should panic for invalid lengths", func() {
			Expect(func() { hash.Fnv("test", 0) }).To(Panic())
			Expect(func() { hash.Fnv("test", -1) }).To(Panic())
			Expect(func() { hash.Fnv("test", 26) }).To(Panic())
		})

		It("should produce different results for different lengths", func() {
			input := "same input"
			result8 := hash.Fnv(input, 8)
			result10 := hash.Fnv(input, 10)
			result13 := hash.Fnv(input, 13)

			// All should be different due to different trimming
			Expect(result8).NotTo(Equal(result10))
			Expect(result8).NotTo(Equal(result13))
			Expect(result10).NotTo(Equal(result13))
		})

		It("should handle unicode correctly", func() {
			result := hash.Fnv("unicode: 你好世界 🌍", 10)
			Expect(result).To(HaveLen(10))
			Expect(result).To(MatchRegexp("^[bcdfghjklmnpqrstvwxz2456789]+$"))
		})

		It("should handle large inputs", func() {
			largeInput := string(make([]byte, 10000))
			result := hash.Fnv(largeInput, 13)
			Expect(result).To(HaveLen(13))
			Expect(result).To(MatchRegexp("^[bcdfghjklmnpqrstvwxz2456789]+$"))
		})

		It("should use 32-bit hash for short lengths (1-7)", func() {
			result := hash.Fnv("test", 6)
			Expect(result).To(HaveLen(6))
			Expect(result).To(MatchRegexp("^[bcdfghjklmnpqrstvwxz2456789]+$"))
		})

		It("should use 64-bit hash for medium lengths (8-13)", func() {
			result := hash.Fnv("test", 10)
			Expect(result).To(HaveLen(10))
			Expect(result).To(MatchRegexp("^[bcdfghjklmnpqrstvwxz2456789]+$"))
		})

		It("should use 128-bit hash for long lengths (14-25)", func() {
			result := hash.Fnv("test", 20)
			Expect(result).To(HaveLen(20))
			Expect(result).To(MatchRegexp("^[bcdfghjklmnpqrstvwxz2456789]+$"))
		})
	})
})

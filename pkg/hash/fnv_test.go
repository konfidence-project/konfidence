package hash_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/konfidence-project/konfidence/pkg/hash"
)

var _ = Describe("FNV Hash Functions", func() {
	Describe("Fnv64", func() {
		It("should return consistent results for the same input", func() {
			input := "test string"
			result1 := hash.Fnv64(input)
			result2 := hash.Fnv64(input)
			Expect(result1).To(Equal(result2))
		})

		It("should return different hashes for different inputs", func() {
			result1 := hash.Fnv64("input1")
			result2 := hash.Fnv64("input2")
			Expect(result1).NotTo(Equal(result2))
		})

		It("should handle empty string", func() {
			result := hash.Fnv64("")
			Expect(result).NotTo(BeEmpty())
			Expect(result).To(MatchRegexp("^[0-9a-z]+$"))
		})

		It("should respect maximum output length of 13 characters", func() {
			// Test with various inputs to ensure output length is reasonable
			inputs := []string{
				"short",
				"a much longer string with lots of content",
				"unicode: 你好世界 🌍",
				string(make([]byte, 10000)), // Large input
			}

			for _, input := range inputs {
				result := hash.Fnv64(input)
				Expect(len(result)).To(BeNumerically("<=", 13),
					"Expected Fnv64 output to be at most 13 characters, got %d for input length %d",
					len(result), len(input))
			}
		})

		It("should produce base36-encoded output", func() {
			result := hash.Fnv64("test")
			// Base36 uses 0-9 and a-z (lowercase)
			Expect(result).To(MatchRegexp("^[0-9a-z]+$"))
		})
	})

	Describe("Fnv128", func() {
		It("should return consistent results for the same input", func() {
			input := "test string"
			result1 := hash.Fnv128(input)
			result2 := hash.Fnv128(input)
			Expect(result1).To(Equal(result2))
		})

		It("should return different hashes for different inputs", func() {
			result1 := hash.Fnv128("input1")
			result2 := hash.Fnv128("input2")
			Expect(result1).NotTo(Equal(result2))
		})

		It("should handle empty string", func() {
			result := hash.Fnv128("")
			Expect(result).NotTo(BeEmpty())
			Expect(result).To(MatchRegexp("^[0-9a-z]+$"))
		})

		It("should respect maximum output length of 25 characters", func() {
			// Test with various inputs to ensure output length is reasonable
			inputs := []string{
				"short",
				"a much longer string with lots of content",
				"unicode: 你好世界 🌍",
				string(make([]byte, 10000)), // Large input
			}

			for _, input := range inputs {
				result := hash.Fnv128(input)
				Expect(len(result)).To(BeNumerically("<=", 25),
					"Expected Fnv128 output to be at most 25 characters, got %d for input length %d",
					len(result), len(input))
			}
		})

		It("should produce base36-encoded output", func() {
			result := hash.Fnv128("test")
			// Base36 uses 0-9 and a-z (lowercase)
			Expect(result).To(MatchRegexp("^[0-9a-z]+$"))
		})

		It("should produce longer hashes than Fnv64 on average", func() {
			input := "sample input for comparison"
			result64 := hash.Fnv64(input)
			result128 := hash.Fnv128(input)

			// 128-bit hashes should generally be longer than 64-bit hashes
			Expect(len(result128)).To(BeNumerically(">=", len(result64)))
		})
	})

	Describe("Fnv64 vs Fnv128 collision resistance", func() {
		It("should produce different hashes between Fnv64 and Fnv128 for the same input", func() {
			input := "collision test"
			result64 := hash.Fnv64(input)
			result128 := hash.Fnv128(input)

			// Different algorithms should produce different outputs
			Expect(result64).NotTo(Equal(result128))
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
				result64 := hash.Fnv64(input)
				result128 := hash.Fnv128(input)

				Expect(result64).To(MatchRegexp("^[0-9a-z]+$"))
				Expect(result128).To(MatchRegexp("^[0-9a-z]+$"))
			}
		})
	})
})

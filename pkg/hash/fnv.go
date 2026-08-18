package hash

import (
	"fmt"
	"hash"
	"hash/fnv"
	"math/big"

	"k8s.io/apimachinery/pkg/util/rand"
)

// Fnv computes an FNV-1a hash of the input content and returns a SafeEncoded
// string trimmed to exactly the specified length.
//
// The output is a base36-encoded string (characters [0-9a-z]) that is always
// exactly 'length' characters long. If the hash is shorter than the requested
// length, it is left-padded with zeros.
//
// Collision Safety Guidelines:
//
// "Safe" means less than 0.1% (1 in 1000) probability of at least one collision
// occurring in your dataset, based on the birthday paradox. These are conservative
// estimates suitable for most non-cryptographic use cases like resource naming,
// caching keys, or short identifiers.
//
//   - 6 chars (~3.1M combinations): Safe for datasets up to ~100 items
//   - 8 chars (~208B combinations): Safe for datasets up to ~75K items
//   - 10 chars (~5.4T combinations): Safe for datasets up to ~7.5M items
//   - 13 chars (~2.5Q combinations): Safe for datasets up to ~100M items
//   - 16 chars (~208Q combinations): Safe for very large datasets (10B+ items)
//
// For higher reliability requirements (e.g., <0.01% collision probability), use
// 2-3 more characters. For critical applications where any collision is
// unacceptable, use cryptographic hashing (SHA-256) instead of FNV.
func Fnv(content string, length int) string {
	if length < 1 || length > 25 {
		panic(fmt.Sprintf("length must be between 1 and 25, got %d", length))
	}

	// Select hash size based on requested length:
	// - 32-bit can produce max 7 chars in base36 (36^7 = 78B > 2^32 = 4.3B)
	// - 64-bit can produce max 13 chars in base36
	// - 128-bit can produce max 25 chars in base36
	var digest hash.Hash
	if length <= 7 {
		digest = fnv.New32a()
	} else if length <= 13 {
		digest = fnv.New64a()
	} else {
		digest = fnv.New128a()
	}

	// It is safe to ignore the error as hash writers never return an error
	_, _ = digest.Write([]byte(content))

	n := new(big.Int).SetBytes(digest.Sum(nil))
	hashStr := n.Text(36)

	// Pad with zeros for numerically small values
	if len(hashStr) < length {
		hashStr = fmt.Sprintf("%0*s", length, hashStr)
	}

	// Apply SafeEncodeString to avoid vowels and potentially offensive words
	hashStr = rand.SafeEncodeString(hashStr)

	// Keep the last 'length' characters (trim from the left)
	return hashStr[len(hashStr)-length:]
}

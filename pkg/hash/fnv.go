package hash

import (
	"hash/fnv"
	"math/big"
	"strconv"
)

// Fnv64 computes a 64-bit FNV-1a hash of the input content and returns it as
// a base36-encoded string.
//
// The FNV-1a algorithm provides good hash distribution for short to medium
// length inputs. The base36 encoding uses characters [0-9a-z], producing
// compact, case-insensitive identifiers.
//
// Maximum output length: 13 characters (base36 encoding of 2^64-1).
func Fnv64(content string) string {
	digest := fnv.New64a()

	// It is safe to ignore the error as hash writers never return an error. See the hash.Hash interface for details.
	_, _ = digest.Write([]byte(content))

	// encode to base36
	return strconv.FormatUint(digest.Sum64(), 36)
}

// Fnv128 computes a 128-bit FNV-1a hash of the input content and returns it as
// a base36-encoded string.
//
// The 128-bit variant provides a larger hash space than Fnv64, significantly
// reducing collision probability for large datasets. Like Fnv64, it uses base36
// encoding for compact, URL-safe output.
//
// Maximum output length: 25 characters (base36 encoding of 2^128-1).
func Fnv128(content string) string {
	digest := fnv.New128a()

	// It is safe to ignore the error as hash writers never return an error. See the hash.Hash interface for details.
	_, _ = digest.Write([]byte(content))

	hashBytes := digest.Sum(nil)
	n := new(big.Int).SetBytes(hashBytes)

	// encode to base36
	return n.Text(36)
}

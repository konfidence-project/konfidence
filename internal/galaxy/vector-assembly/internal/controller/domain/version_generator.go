package domain

import "time"

// VectorVersionGenerator generates version strings for vectors.
type VectorVersionGenerator interface {
	// Generate creates a new version string for a vector.
	Generate() string
}

// VectorVersionGeneratorFunc is a functional adapter that allows using a simple function as a VectorVersionGenerator.
// If f is a function using the signature func() string, VectorVersionGeneratorFunc(f) is a VectorVersionGenerator.
type VectorVersionGeneratorFunc func() string

func (g VectorVersionGeneratorFunc) Generate() string {
	return g()
}

// TimestampVectorVersionGenerator generates version strings based on the current UTC timestamp in the format
// "YYYY.M.D-HHMMSSmmmZ" (e.g., "2024.6.10-153045123Z").
var TimestampVectorVersionGenerator = VectorVersionGeneratorFunc(func() string {
	return time.Now().UTC().Format("2006.1.2-150405000Z")
})

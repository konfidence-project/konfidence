package vector

import "time"

// VersionGenerator generates version strings for vectors.
type VersionGenerator interface {
	// Generate creates a new version string for a vector.
	Generate() string
}

// VersionGeneratorFunc is a functional adapter that allows using a simple function as a VersionGenerator.
// If f is a function using the signature func() string, VersionGeneratorFunc(f) is a VersionGenerator.
type VersionGeneratorFunc func() string

func (g VersionGeneratorFunc) Generate() string {
	return g()
}

// TimestampVectorVersionGenerator generates version strings based on the current UTC timestamp in the format
// "YYYY.M.D-HHMMSSmmmZ" (e.g., "2024.6.10-153045123Z").
var TimestampVectorVersionGenerator = VersionGeneratorFunc(func() string {
	return time.Now().UTC().Format("2006.1.2-150405000Z")
})

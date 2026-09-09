// Package build exposes metadata about the running binary: the release
// version, git commit, and build date. The vars should be set at link time via
// -ldflags and fall back to placeholder values under a plain `go build`.
package build

import "runtime"

// These are set at link time via -ldflags and fall back to the placeholder
// values below under a plain `go build`.
var (
	Version   = "dev"
	GitCommit = "unknown"
	Date      = "unknown"
)

// Info describes the running build: the linked-in release metadata plus the Go
// toolchain and target platform it was compiled for.
type Info struct {
	Version   string `json:"version" yaml:"version"`
	Commit    string `json:"commit" yaml:"commit"`
	GoVersion string `json:"goVersion" yaml:"goVersion"`
	Platform  string `json:"platform" yaml:"platform"`
	Date      string `json:"built" yaml:"built"`
}

// Current snapshots the injected build vars together with the runtime's Go
// version and target platform.
func Current() Info {
	return Info{
		Version:   Version,
		Commit:    GitCommit,
		GoVersion: runtime.Version(),
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
		Date:      Date,
	}
}

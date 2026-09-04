package version

import "testing"

func TestIsRelease(t *testing.T) {
	cases := []struct {
		version string
		want    bool
	}{
		{"v0.4.0", true},
		{"0.0.1-alpha.1", true},
		{"1.2.3", true},
		{"dev", false},       // default ldflags value
		{"main", false},      // KDEN_GIT_REF branch build
		{"my-branch", false}, // KDEN_GIT_REF branch build
		{"abc1234", false},   // KDEN_GIT_REF sha build
		{"", false},
	}
	for _, c := range cases {
		if got := isRelease(c.version); got != c.want {
			t.Errorf("isRelease(%q) = %v, want %v", c.version, got, c.want)
		}
	}
}

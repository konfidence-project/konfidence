package compref

import (
	"errors"
	"fmt"
	"regexp"

	"ocm.software/open-component-model/bindings/go/oci/compref"
)

// ErrInvalidComponentReference is returned when a component reference fails validation.
var ErrInvalidComponentReference = errors.New("invalid component reference")

const (
	// VersionRegex is the regular expression used by upstream to validate semantic versioning.
	// It allows for optional "v" prefix and supports pre-release and build metadata.
	VersionRegex = compref.VersionRegex
)

var (
	versionRegex = regexp.MustCompile(VersionRegex)
)

// VersionValidationMode defines how strictly version strings are validated.
type VersionValidationMode int

const (
	// VersionValidationPermissive allows any OCI tag as a version, including both
	// semantic versions and aliases like "latest", "main", "stable", etc.
	// This is the default mode and supports the full flexibility of OCI tagging.
	VersionValidationPermissive VersionValidationMode = iota

	// VersionValidationSemverOnly requires versions to match semantic versioning rules.
	// Only versions matching VersionRegex are accepted. Aliases like "latest" or "main"
	// will be rejected.
	VersionValidationSemverOnly

	// VersionValidationAliasOnly requires versions to be aliases (non-semver tags).
	// Semantic versions will be rejected.
	VersionValidationAliasOnly

	// VersionValidationNoVersion requires the reference to carry no version at all
	// (repository and component only). A non-empty version is rejected. Use this for
	// references that name a component whose version is assigned by the system rather
	// than the user, e.g. a VectorTemplate uploadTarget.
	VersionValidationNoVersion
)

// ParseOption configures the behavior of Parse.
type ParseOption func(*parseConfig)

type parseConfig struct {
	validate    bool
	versionMode VersionValidationMode
}

// WithoutValidation skips validation when parsing.
func WithoutValidation() ParseOption {
	return func(cfg *parseConfig) {
		cfg.validate = false
	}
}

// WithVersionValidation configures how version strings are validated during parsing.
//
// By default, Parse uses VersionValidationPermissive, which accepts any OCI tag.
// Use this option to enforce stricter validation:
//
//   - VersionValidationSemverOnly: Only accept semantic versions (e.g., "1.0.0", "v2.1.0-rc.1")
//   - VersionValidationAliasOnly: Only accept non-semver tags (e.g., "latest", "main", "staging")
//
// Examples:
//
//	// Reject non-semver versions like "latest" or "main"
//	ref, err := Parse("ghcr.io/org/components//github.com/org/app:1.0.0",
//	    WithVersionValidation(VersionValidationSemverOnly))
//
//	// Reject semver versions, only allow aliases
//	ref, err := Parse("ghcr.io/org/components//github.com/org/app:main",
//	    WithVersionValidation(VersionValidationAliasOnly))
func WithVersionValidation(mode VersionValidationMode) ParseOption {
	return func(cfg *parseConfig) {
		cfg.versionMode = mode
	}
}

// Parse parses an OCM component reference string into a structured Ref.
//
// This function wraps the upstream compref.Parse with IgnoreSemverCompatibility enabled,
// allowing arbitrary OCI tags to be used as component versions. This supports versioning
// strategies beyond strict semantic versioning. Especially support for alias retrieval.
//
// By default, Parse validates the resulting reference with VersionValidationPermissive mode.
// Use WithoutValidation to skip validation entirely, or WithVersionValidation to enforce
// stricter validation modes.
//
// # Format
//
//	<repository>//<component>:<version>
//
// Examples:
//
//	Parse("ghcr.io/org/components//github.com/org/app:latest")
//	Parse("ghcr.io/org/components//github.com/org/app:main")
//	Parse("ghcr.io/org/components//github.com/org/app:1.0.0")
//	Parse("ghcr.io/org/components//github.com/org/app", WithoutValidation())
//	Parse("ghcr.io/org/components//github.com/org/app:1.0.0", WithVersionValidation(VersionValidationSemverOnly))
//	Parse("ghcr.io/org/components//github.com/org/app:main", WithVersionValidation(VersionValidationAliasOnly))
//
// Returns ErrInvalidComponentReference if parsing or validation fails.
func Parse(ref string, opts ...ParseOption) (*compref.Ref, error) {
	cfg := parseConfig{
		validate:    true,
		versionMode: VersionValidationPermissive,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	parsed, err := compref.Parse(ref, compref.IgnoreSemverCompatibility())
	if err != nil {
		return nil, errors.Join(
			ErrInvalidComponentReference,
			fmt.Errorf("failed to parse component reference %q: %w", ref, err),
		)
	}

	if cfg.validate {
		if err := Validate(*parsed, []ValidationOption{WithVersionValidationMode(cfg.versionMode)}...); err != nil {
			return nil, fmt.Errorf("reference validation failed after parsing: %w", err)
		}
	}

	return parsed, nil
}

// Validate checks if a component reference is valid.
//
// Upstream compref.Parse only requires the repository and component to be set.
// Validate will also check that a non-empty version is set on the reference.
//
// By default, Validate uses VersionValidationPermissive mode, accepting any OCI tag.
// Use WithVersionValidationMode to enforce stricter validation:
//
//   - VersionValidationSemverOnly: Reject aliases, only accept semantic versions
//   - VersionValidationAliasOnly: Reject semantic versions, only accept aliases
//
// Examples:
//
//	// Accept any version string
//	err := Validate(ref)
//
//	// Only accept semantic versions
//	err := Validate(ref, WithVersionValidationMode(VersionValidationSemverOnly))
//
//	// Only accept alias tags (non-semver)
//	err := Validate(ref, WithVersionValidationMode(VersionValidationAliasOnly))
//
// Returns ErrInvalidComponentReference if validation fails.
func Validate(ref compref.Ref, opts ...ValidationOption) error {
	cfg := validationConfig{
		versionMode: VersionValidationPermissive,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.versionMode == VersionValidationNoVersion {
		if ref.Version != "" {
			return fmt.Errorf(
				"%w: component reference %q must not carry a version", ErrInvalidComponentReference, ref)
		}
		return nil
	}

	if ref.Version == "" {
		return fmt.Errorf("%w: component reference %q is missing version", ErrInvalidComponentReference, ref)
	}

	switch cfg.versionMode {
	case VersionValidationSemverOnly:
		if isSemver := versionRegex.MatchString(ref.Version); !isSemver {
			return fmt.Errorf(
				"%w: version %q is not a valid semantic version (must match %s)",
				ErrInvalidComponentReference,
				ref.Version,
				VersionRegex,
			)
		}
	case VersionValidationAliasOnly:
		if isSemver := versionRegex.MatchString(ref.Version); isSemver {
			return fmt.Errorf(
				"%w: version %q is a semantic version but only aliases are allowed",
				ErrInvalidComponentReference,
				ref.Version,
			)
		}
	case VersionValidationPermissive:
		// Accept both semver and aliases - no additional checks needed
	case VersionValidationNoVersion:
		// handled above
	}

	return nil
}

// ValidationOption configures the behavior of Validate.
type ValidationOption func(*validationConfig)

type validationConfig struct {
	versionMode VersionValidationMode
}

// WithVersionValidationMode configures how version strings are validated.
// Use this with Validate to enforce version format restrictions.
func WithVersionValidationMode(mode VersionValidationMode) ValidationOption {
	return func(cfg *validationConfig) {
		cfg.versionMode = mode
	}
}

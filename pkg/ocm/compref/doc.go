// Package compref provides parsing of Open Component Model (OCM) component references
// with support for OCI tag aliases.
//
// # Overview
//
// This package wraps the upstream compref parser to handle OCI tags as component versions.
// Unlike the upstream default, it ignores semantic versioning compatibility checks, allowing
// arbitrary OCI tags (like "latest", "stable", "main", custom aliases) to be used as
// component versions.
//
// # Quick Start
//
//	import "github.com/konfidence-project/konfidence/pkg/ocm/compref"
//
//	// Parse references with OCI tag aliases (validates by default)
//	ref, err := compref.Parse("ghcr.io/org/components//github.com/org/app:latest")
//	ref, err := compref.Parse("ghcr.io/org/components//github.com/org/app:main")
//	ref, err := compref.Parse("ghcr.io/org/components//github.com/org/app:stable")
//	ref, err := compref.Parse("ghcr.io/org/components//github.com/org/app:1.0.0")
//
//	// Skip validation if needed
//	ref, err := compref.Parse("ghcr.io/org/components//github.com/org/app", compref.WithoutValidation())
//
//	// Enforce semantic version only (reject aliases)
//	ref, err := compref.Parse("ghcr.io/org/components//github.com/org/app:1.0.0",
//		compref.WithVersionValidation(compref.VersionValidationSemverOnly))
//
//	// Enforce aliases only (reject semantic versions)
//	ref, err := compref.Parse("ghcr.io/org/components//github.com/org/app:main",
//		compref.WithVersionValidation(compref.VersionValidationAliasOnly))
//
//	// Validate a reference manually
//	if err := compref.Validate(*ref); err != nil {
//		// handle invalid reference (e.g., missing version)
//	}
//
//	// Validate with version mode restriction
//	err = compref.Validate(*ref,
//		compref.WithVersionValidationMode(compref.VersionValidationSemverOnly))
//	if err != nil {
//		// handle invalid version format
//	}
//
// # Reference Format
//
// Component references follow the format:
//
//	<repository>//<component>:<version>
//
// Where version can be any valid OCI tag, not just semver strings.
//
// By default, Parse validates references. This ensures references are complete and actionable.
// Use WithoutValidation() to skip this check.
//
// # Version Validation Modes
//
// Three validation modes control how version strings are validated:
//
//   - VersionValidationPermissive (default): Accept both semantic versions and aliases.
//
//   - VersionValidationSemverOnly: Only accept semantic versions matching the VersionRegex
//     (e.g., "1.0.0", "v2.1.0-rc.1+build.123"). Reject aliases like "latest" or "main".
//
//   - VersionValidationAliasOnly: Only accept non-semver OCI tags (e.g., "latest", "main",
//     "production"). Reject semantic versions.
//
// # Why Ignore Semver Compatibility?
//
// OCI registries support arbitrary tags beyond semantic versions. Common patterns include:
//   - Branch aliases: "main", "develop", "feature-xyz"
//   - Environment aliases: "production", "staging", "canary"
//   - Stability aliases: "latest", "stable", "edge"
//   - Custom aliases: "nightly-2024-03-15", "release-candidate"
//
// By disabling strict semver validation in the underlying parser, this package allows OCM
// component references to leverage the full flexibility of OCI tagging for versioning
// strategies beyond semver. You can then enforce stricter validation at the application
// layer using the validation modes.
//
// # See Also
//
//   - ocm/repository: Read and write component descriptors using parsed references
//   - Upstream: ocm.software/open-component-model/bindings/go/oci/compref
package compref

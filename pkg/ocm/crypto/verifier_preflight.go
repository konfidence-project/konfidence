package crypto

import (
	"context"
	"fmt"
	"strings"

	"ocm.software/open-component-model/bindings/go/credentials"
	ocm "ocm.software/open-component-model/bindings/go/descriptor/runtime"
)

var _ Verifier = (*preFlightVerifier)(nil)

// preFlightVerifier is a Verifier decorator that validates the inputs once, at
// the top of the chain, before any fan-out, caching, or crypto:
//   - the SignatureSpecs (non-empty names, no duplicates, valid issuer pins)
//   - that every descriptor is safely digestible
//
// Both are structural preconditions of the whole descriptor batch, independent
// of any single (spec, desc) cell — so they run here exactly once, not
// per-cell in the inner layers.
type preFlightVerifier struct {
	inner Verifier
}

func newPreFlightVerifier(inner Verifier) *preFlightVerifier {
	return &preFlightVerifier{inner: inner}
}

func (v *preFlightVerifier) Verify(ctx context.Context, resolver credentials.Resolver, specs []SignatureSpec, descs []*ocm.Descriptor) error {
	if len(specs) == 0 || len(descs) == 0 {
		return nil
	}
	if err := specPreFlightSanityCheck(specs); err != nil {
		return fmt.Errorf("ocm descriptor verification failed for the cr specs provided: %w", err)
	}
	for _, desc := range descs {
		if err := isSafelyDigestible(&desc.Component); err != nil {
			return fmt.Errorf("ocm descriptor verification failed: descriptor is not safely digestible: %w", err)
		}
	}
	return v.inner.Verify(ctx, resolver, specs, descs)
}

// specPreFlightSanityCheck validates SignatureSpecs: non-empty names, no
// duplicate names, and no spec with a non-nil empty issuer pin.
func specPreFlightSanityCheck(specs []SignatureSpec) error {
	seen := make(map[string]struct{}, len(specs))
	for _, spec := range specs {
		if strings.TrimSpace(spec.Name) == "" {
			return fmt.Errorf("signature names cannot be empty or whitespace")
		}
		if _, exists := seen[spec.Name]; exists {
			return fmt.Errorf("duplicate signature name detected: %q", spec.Name)
		}
		seen[spec.Name] = struct{}{}
		if spec.Issuer != nil && *spec.Issuer == "" {
			return fmt.Errorf("issuer pin for %q must not be empty; use nil to disable issuer pinning", spec.Name)
		}
	}
	return nil
}

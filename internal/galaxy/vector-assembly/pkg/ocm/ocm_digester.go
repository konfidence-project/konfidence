package ocm

import (
	"context"
	"crypto"
	"fmt"
	"log/slog"

	norm "ocm.software/open-component-model/bindings/go/descriptor/normalisation/json/v4alpha1"
	ocmDescriptor "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	"ocm.software/open-component-model/bindings/go/signing"
)

var (
	_ Digester = (*ocmDigester)(nil)
)

// Digester is an interface for generating digests for OCM descriptors.
type Digester interface {
	// GenerateDigest generates a digest for the given OCM descriptor. If digest generation fails, a non-nil error is returned.
	GenerateDigest(ctx context.Context, desc *ocmDescriptor.Descriptor) (*ocmDescriptor.Digest, error)
	// GetHashAlgorithm returns the hash algorithm used for generating digests.
	GetHashAlgorithm() crypto.Hash
	// GetNormalisationAlgorithm returns the normalisation algorithm used for generating digests.
	GetNormalisationAlgorithm() string
}

// ocmDigester is the default implementation of the Digester interface for generating digests for OCM descriptors.
type ocmDigester struct{}

func (d ocmDigester) GenerateDigest(ctx context.Context, desc *ocmDescriptor.Descriptor) (*ocmDescriptor.Digest, error) {
	if err := signing.IsSafelyDigestible(&desc.Component); err != nil {
		return nil, fmt.Errorf("unable to generate digest for ocm descriptor, descriptor is not safely digestible: %w", err)
	}
	dig, err := signing.GenerateDigest(ctx, desc, slog.Default(), d.GetNormalisationAlgorithm(), d.GetHashAlgorithm().String())
	if err != nil {
		return nil, fmt.Errorf("unable to generate digest for ocm descriptor, generating digest failed: %w", err)
	}
	return dig, nil
}

func (d ocmDigester) GetHashAlgorithm() crypto.Hash {
	return crypto.SHA256
}

func (d ocmDigester) GetNormalisationAlgorithm() string {
	return norm.Algorithm
}

func newOcmDigester() Digester {
	return ocmDigester{}
}

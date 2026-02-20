package crypto

import (
	"context"
	"crypto"
	"fmt"
	"log/slog"

	"github.com/go-logr/logr"
	norm "ocm.software/open-component-model/bindings/go/descriptor/normalisation/json/v4alpha1"
	"ocm.software/open-component-model/bindings/go/descriptor/runtime"
	"ocm.software/open-component-model/bindings/go/signing"
)

var (
	_                  Digester = (*DefaultDigester)(nil)
	isSafelyDigestible          = signing.IsSafelyDigestible
	generateDigest              = signing.GenerateDigest
)

// Digester is an interface for generating digests for OCM descriptors.
type Digester interface {
	// GenerateDigest generates a digest for the given OCM descriptor.
	// If digest generation fails, a non-nil error is returned.
	GenerateDigest(ctx context.Context, desc *runtime.Descriptor) (*runtime.Digest, error)
	// GetHashAlgorithm returns the hash algorithm used for generating digests.
	GetHashAlgorithm() crypto.Hash
	// GetNormalisationAlgorithm returns the normalisation algorithm used for generating digests.
	GetNormalisationAlgorithm() string
}

// DefaultDigester is the default implementation of the Digester interface for generating digests for OCM descriptors.
// It uses the SHA256 hash algorithm and the jsonNormalisation/v4alpha1 normalization algorithm.
// It also ensures that the descriptor is safely digestible before generating the digest.
type DefaultDigester struct {
	// since this logger is only used for handing it to the signing.GenerateDigest function
	// we store the slog.Logger directly
	log *slog.Logger
}

func (d DefaultDigester) GenerateDigest(ctx context.Context, desc *runtime.Descriptor) (*runtime.Digest, error) {
	if err := isSafelyDigestible(&desc.Component); err != nil {
		return nil, fmt.Errorf("unable to generate digest for ocm descriptor, descriptor is not safely digestible: %w", err)
	}
	dig, err := generateDigest(ctx, desc, d.log, d.GetNormalisationAlgorithm(), d.GetHashAlgorithm().String())
	if err != nil {
		return nil, fmt.Errorf("unable to generate digest for ocm descriptor, generating digest failed: %w", err)
	}
	return dig, nil
}

func (d DefaultDigester) GetHashAlgorithm() crypto.Hash {
	return crypto.SHA256
}

func (d DefaultDigester) GetNormalisationAlgorithm() string {
	return norm.Algorithm
}

func NewDefaultDigester(log logr.Logger) DefaultDigester {
	return DefaultDigester{
		log: slog.New(logr.ToSlogHandler(log.WithName("ocm-digester"))),
	}
}

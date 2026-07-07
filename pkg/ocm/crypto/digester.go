package crypto

//go:generate go run go.uber.org/mock/mockgen -destination=internal/mocks/mock_digester.go -package=mocks github.com/konfidence-project/konfidence/pkg/ocm/crypto Digester

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
	_                  Digester = (*ConfigurableDigester)(nil)
	isSafelyDigestible          = signing.IsSafelyDigestible
	generateDigest              = signing.GenerateDigest
)

// Digester is an interface for generating digests for OCM descriptors.
type Digester interface {
	// GenerateDigest generates a digest for the given OCM descriptor.
	// If digest generation fails, a non-nil error is returned.
	GenerateDigest(ctx context.Context, desc *runtime.Descriptor) (*runtime.Digest, error)
	// GetHashAlgorithm returns the hash algorithm used for generating digests.
	GetHashAlgorithm() string
	// GetNormalisationAlgorithm returns the normalisation algorithm used for generating digests.
	GetNormalisationAlgorithm() string
}

// ConfigurableDigester is a configurable implementation of the Digester interface for
// generating digests for OCM descriptors.
type ConfigurableDigester struct {
	// since this logger is only used for handing it to the signing.GenerateDigest function
	// we store the slog.Logger directly
	log                    *slog.Logger
	hashAlgorithm          string
	normalisationAlgorithm string
}

func (d ConfigurableDigester) GenerateDigest(ctx context.Context, desc *runtime.Descriptor) (*runtime.Digest, error) {
	if err := isSafelyDigestible(&desc.Component); err != nil {
		return nil, fmt.Errorf("unable to generate digest for ocm descriptor, descriptor is not safely digestible: %w", err)
	}
	dig, err := generateDigest(ctx, desc, d.log, d.normalisationAlgorithm, d.hashAlgorithm)
	if err != nil {
		return nil, fmt.Errorf("unable to generate digest for ocm descriptor, generating digest failed: %w", err)
	}
	return dig, nil
}

func (d ConfigurableDigester) GetHashAlgorithm() string {
	return d.hashAlgorithm
}

func (d ConfigurableDigester) GetNormalisationAlgorithm() string {
	return d.normalisationAlgorithm
}

// NewDigester creates a new configurable digester with the given options.
// If no hash algorithm is provided crypto.SHA256.String() is used as default.
// If no normalisation algorithm is provided norm.Algorithm is used as default.
func NewDigester(options ...DigestOption) ConfigurableDigester {
	a := ConfigurableDigester{}
	for _, opt := range options {
		opt(&a)
	}
	if a.hashAlgorithm == "" {
		a.hashAlgorithm = crypto.SHA256.String()
	}
	if a.normalisationAlgorithm == "" {
		a.normalisationAlgorithm = norm.Algorithm
	}
	return a
}

func NewDefaultDigester(log logr.Logger) ConfigurableDigester {
	return NewDigester(WithLog(log))
}

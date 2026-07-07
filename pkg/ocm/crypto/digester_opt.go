package crypto

import (
	"log/slog"

	"github.com/go-logr/logr"
)

// DigestOption is a functional option for configuring a digest.
type DigestOption func(*ConfigurableDigester)

// WithLog sets the digester log.
func WithLog(log logr.Logger) DigestOption {
	return func(d *ConfigurableDigester) {
		d.log = slog.New(logr.ToSlogHandler(log.WithName("ocm-digester")))
	}
}

// WithHashAlgorithm sets the hash algorithm used for calculating the digest.
// If no algorithm is provided crypto.SHA256.String() is used as default.
func WithHashAlgorithm(hashAlgorithm string) DigestOption {
	return func(d *ConfigurableDigester) {
		d.hashAlgorithm = hashAlgorithm
	}
}

// WithNormalizationAlgorithm sets the normalization algorithm used for calculating the digest.
// If no algorithm is provided norm.Algorithm is used as default.
func WithNormalizationAlgorithm(normalizationAlgorithm string) DigestOption {
	return func(d *ConfigurableDigester) {
		d.normalisationAlgorithm = normalizationAlgorithm
	}
}

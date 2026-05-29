package crypto

//go:generate go run go.uber.org/mock/mockgen -destination=internal/mocks/mock_ocm_verifier.go -package=mocks ocm.software/open-component-model/bindings/go/signing Verifier

import (
	"context"
	"fmt"
	"log/slog"
	sysruntime "runtime"
	"slices"

	"github.com/go-logr/logr"
	"golang.org/x/sync/errgroup"
	ocm "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	rsahandler "ocm.software/open-component-model/bindings/go/rsa/signing/handler"
	"ocm.software/open-component-model/bindings/go/runtime"
	"ocm.software/open-component-model/bindings/go/signing"
)

var (
	_                             Verifier = (*RSAVerifier)(nil)
	_                             Verifier = (*NoopVerifier)(nil)
	verifyDigestMatchesDescriptor          = signing.VerifyDigestMatchesDescriptor
)

// Verifier is an interface for verifying OCM descriptors.
type Verifier interface {
	// Verify verifies the signatures of the given OCM descriptors.
	// If any verification fails, an error is returned.
	Verify(ctx context.Context, descs ...*ocm.Descriptor) error
}

// RSAVerifier is the default implementation of the Verifier interface for verifying OCM descriptors.
// If an RSACredentialProvider is configured, it will be used to provide additional trust anchors.
// If no provider is configured, the RSAVerifier will only use the system trust store for verification.
type RSAVerifier struct {
	log              logr.Logger
	provider         RSACredentialProvider
	rsaVerifier      signing.Verifier
	targetSignatures []string
}

// RSAVerifierOption configures an RSAVerifier.
type RSAVerifierOption func(*RSAVerifier)

// WithLogger sets the logger for the verifier.
// The constructor automatically appends signature names as log values.
func WithLogger(log logr.Logger) RSAVerifierOption {
	return func(v *RSAVerifier) {
		v.log = log
	}
}

// WithVerifierLogger decorates the logger with the standard verifier name "ocm-rsa-verifier".
// The constructor automatically appends signature names as log values.
func WithVerifierLogger(log logr.Logger) RSAVerifierOption {
	return func(v *RSAVerifier) {
		v.log = log.WithName("ocm-rsa-verifier")
	}
}

// WithCredentialProvider sets the credential provider for additional trust anchors.
// The caller is responsible for starting and managing the provider's lifecycle.
func WithCredentialProvider(provider RSACredentialProvider) RSAVerifierOption {
	return func(v *RSAVerifier) {
		v.provider = provider
	}
}

func defaultRSAVerifierOptions() *RSAVerifier {
	return &RSAVerifier{
		log: logr.Discard(),
	}
}

func (o *RSAVerifier) Verify(ctx context.Context, descs ...*ocm.Descriptor) error {
	var (
		creds map[string]string
		err   error
	)
	if o.provider != nil {
		creds, err = o.provider.Get(ctx)
		if err != nil {
			return fmt.Errorf("get credentials from provider: %w", err)
		}
	}
	if len(descs) == 1 {
		return o.verify(ctx, creds, descs[0])
	}
	verifierPool, ctx2 := errgroup.WithContext(ctx)
	verifierPool.SetLimit(min(sysruntime.GOMAXPROCS(0), len(descs))) // no oversubscription on CPU bound verification tasks
	for _, t := range descs {
		verifierPool.Go(func() error { return o.verify(ctx2, creds, t) })
	}
	return verifierPool.Wait()
}

func (o *RSAVerifier) verify(ctx context.Context, creds map[string]string, desc *ocm.Descriptor) error {
	if err := isSafelyDigestible(&desc.Component); err != nil {
		return fmt.Errorf("ocm descriptor verification failed: descriptor is not safely digestible: %w", err)
	}
	toVerify := make([]ocm.Signature, 0, len(o.targetSignatures))
	for _, targetSignature := range o.targetSignatures {
		if idx := slices.IndexFunc(
			desc.Signatures,
			func(sig ocm.Signature) bool { return sig.Name == targetSignature }); idx == -1 {
			return fmt.Errorf(
				"ocm descriptor verification failed: signature with name %q not found in descriptor",
				targetSignature)
		} else {
			toVerify = append(toVerify, desc.Signatures[idx])
		}
	}
	for _, sig := range toVerify {
		if err := verifyDigestMatchesDescriptor(ctx, desc, sig, slog.New(logr.ToSlogHandler(o.log))); err != nil {
			return fmt.Errorf(
				"ocm descriptor verification failed: digest verification failed for signature with name %q: %w",
				sig.Name, err)
		}
		if err := o.rsaVerifier.Verify(ctx, sig, nil, creds); err != nil {
			return fmt.Errorf(
				"ocm descriptor verification failed: signature verification failed for signature with name %q: %w",
				sig.Name, err)
		}
	}
	return nil
}

// NewRSAVerifier creates a new RSAVerifier instance.
// At least one signature name must be provided.
func NewRSAVerifier(signatures []string, opts ...RSAVerifierOption) (*RSAVerifier, error) {
	if err := signaturePreFlightSanityCheck(signatures); err != nil {
		return nil, fmt.Errorf("create verifier: %w", err)
	}
	rsaHandler, err := rsahandler.New(runtime.NewScheme(), true) // load system roots
	if err != nil {
		return nil, fmt.Errorf("create rsa handler: %w", err)
	}
	v := defaultRSAVerifierOptions()
	for _, opt := range opts {
		opt(v)
	}
	v.log = v.log.WithValues("signatures", fmt.Sprintf("%v", signatures))
	v.rsaVerifier = rsaHandler
	v.targetSignatures = signatures
	return v, nil
}

// NoopVerifier is a Verifier implementation that does not perform any verification and returns nil for all operations.
// It's the goto way to disable verification.
type NoopVerifier struct{}

func (n NoopVerifier) Verify(ctx context.Context, descs ...*ocm.Descriptor) error {
	return nil
}

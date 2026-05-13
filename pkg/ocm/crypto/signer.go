package crypto

//go:generate go run go.uber.org/mock/mockgen -destination=internal/mocks/mock_ocm_signer.go -package=mocks ocm.software/open-component-model/bindings/go/signing Signer

import (
	"context"
	"fmt"
	sysRuntime "runtime"
	"slices"

	"github.com/go-logr/logr"
	"golang.org/x/sync/errgroup"
	ocmDescriptor "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	rsahandler "ocm.software/open-component-model/bindings/go/rsa/signing/handler"
	rsav1alpha1 "ocm.software/open-component-model/bindings/go/rsa/signing/v1alpha1"
	"ocm.software/open-component-model/bindings/go/runtime"
	"ocm.software/open-component-model/bindings/go/signing"
)

const (
	signingAlgorithm = rsav1alpha1.AlgorithmRSASSAPSS
)

var (
	_ Signer = (*RSASigner)(nil)
	_ Signer = (*NoopSigner)(nil)
)

// Signer is an interface for signing OCM descriptors.
type Signer interface {
	// Sign signs the given OCM descriptor and adds the signatures to the descriptor's signatures.
	// If signing fails or the signature already exists, a non-nil error is returned.
	Sign(ctx context.Context, desc *ocmDescriptor.Descriptor) error
}

// RSASigner is the default implementation of the Signer interface for signing OCM descriptors.
// It uses the rsav1alpha1.AlgorithmRSASSAPSS RSA Probabilistic Signature Scheme.
type RSASigner struct {
	log              logr.Logger
	provider         RSACredentialProvider
	rsaSigner        signing.Signer
	targetSignatures []string
	rsaConfig        *rsav1alpha1.Config
	digester         Digester
}

// RSASignerOption configures an RSASigner.
type RSASignerOption func(*RSASigner)

// WithSignerLogger sets the logger for the signer.
// The constructor automatically appends signature names as log values.
func WithSignerLogger(log logr.Logger) RSASignerOption {
	return func(s *RSASigner) {
		s.log = log
	}
}

// WithNamedSignerLogger decorates the logger with the standard signer name "ocm-rsa-signer".
// The constructor automatically appends signature names as log values.
func WithNamedSignerLogger(log logr.Logger) RSASignerOption {
	return func(s *RSASigner) {
		s.log = log.WithName("ocm-rsa-signer")
	}
}

// WithDigester sets the digester for the signer.
// If not provided, a digester with default values is used.
func WithDigester(digester Digester) RSASignerOption {
	return func(s *RSASigner) {
		s.digester = digester
	}
}

func defaultRSASignerOptions() *RSASigner {
	return &RSASigner{
		log:      logr.Discard(),
		digester: NewDigester(),
	}
}

func (s *RSASigner) Sign(ctx context.Context, desc *ocmDescriptor.Descriptor) error {
	for _, signatureName := range s.targetSignatures {
		if slices.ContainsFunc(desc.Signatures, func(sig ocmDescriptor.Signature) bool { return sig.Name == signatureName }) {
			return fmt.Errorf("signature with name %q already exists", signatureName)
		}
	}
	dig, err := s.digester.GenerateDigest(ctx, desc)
	if err != nil {
		return fmt.Errorf("generate digest: %w", err)
	}
	creds, err := s.provider.Get(ctx)
	if err != nil {
		return fmt.Errorf("get credentials from provider: %w", err)
	}
	if creds == nil {
		return fmt.Errorf("signing credentials are not available")
	}
	// results is an all or nothing buffer for the signing results
	results := make([]ocmDescriptor.Signature, len(s.targetSignatures))
	if len(s.targetSignatures) == 1 {
		if err := s.sign(ctx, results, 0, creds, dig, s.targetSignatures[0]); err != nil {
			return err
		}
	} else {
		signerPool, ctx2 := errgroup.WithContext(ctx)
		// no oversubscription on CPU bound signing tasks
		signerPool.SetLimit(min(sysRuntime.GOMAXPROCS(0), len(s.targetSignatures)))
		for idx, sig := range s.targetSignatures {
			signerPool.Go(func() error { return s.sign(ctx2, results, idx, creds, dig, sig) })
		}
		if err := signerPool.Wait(); err != nil {
			return fmt.Errorf("signing failed: %w", err)
		}
	}
	desc.Signatures = append(desc.Signatures, results...)
	return nil
}

func (s *RSASigner) sign(
	ctx context.Context,
	results []ocmDescriptor.Signature,
	idx int,
	creds map[string]string,
	dig *ocmDescriptor.Digest,
	signatureName string) error {
	signatureInfo, err := s.rsaSigner.Sign(ctx, *dig, s.rsaConfig, creds)
	if err != nil {
		return fmt.Errorf("sign %q: %w", signatureName, err)
	}
	results[idx] = ocmDescriptor.Signature{
		Name:      signatureName,
		Digest:    *dig,
		Signature: signatureInfo,
	}
	return nil
}

// NewRSASigner creates a new RSASigner instance.
// At least one signature name must be provided.
func NewRSASigner(
	provider RSACredentialProvider,
	signatures []string,
	opts ...RSASignerOption,
) (*RSASigner, error) {
	if provider == nil {
		return nil, fmt.Errorf("create signer: credential provider is required")
	}
	if err := signaturePreFlightSanityCheck(signatures); err != nil {
		return nil, fmt.Errorf("create signer: %w", err)
	}
	rsaHandler, err := rsahandler.New(runtime.NewScheme(), false)
	if err != nil {
		return nil, fmt.Errorf("create rsa handler: %w", err)
	}
	s := defaultRSASignerOptions()
	for _, opt := range opts {
		opt(s)
	}
	s.log = s.log.WithValues("signatures", fmt.Sprintf("%v", signatures))
	s.rsaSigner = rsaHandler
	s.targetSignatures = signatures
	s.provider = provider
	s.rsaConfig = &rsav1alpha1.Config{
		Type:                    runtime.NewVersionedType(rsav1alpha1.ConfigType, rsav1alpha1.Version),
		SignatureAlgorithm:      signingAlgorithm,
		SignatureEncodingPolicy: rsav1alpha1.SignatureEncodingPolicyPEM,
	}
	return s, nil
}

// NoopSigner is a Signer implementation that does not perform any signing and returns nil for all operations.
// It's the goto way to disable signing.
type NoopSigner struct{}

func (n NoopSigner) Sign(ctx context.Context, desc *ocmDescriptor.Descriptor) error {
	return nil
}

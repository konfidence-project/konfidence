package crypto

//go:generate go run go.uber.org/mock/mockgen -destination=internal/mocks/mock_ocm_signer.go -package=mocks ocm.software/open-component-model/bindings/go/signing Signer

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"golang.org/x/sync/errgroup"
	"ocm.software/open-component-model/bindings/go/credentials"
	ocmdescriptor "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	rsahandler "ocm.software/open-component-model/bindings/go/rsa/signing/handler"
	rsav1alpha1 "ocm.software/open-component-model/bindings/go/rsa/signing/v1alpha1"
	"ocm.software/open-component-model/bindings/go/runtime"
	"ocm.software/open-component-model/bindings/go/signing"
)

var (
	_ Signer = (*ocmSigner)(nil)
)

// Signer is an interface for signing OCM descriptors.
type Signer interface {
	// Sign signs the given OCM descriptor and adds signatures to the descriptor's signatures.
	// If signing fails or the signature already exists, a non-nil error is returned.
	// In case of multiple signatures being applied, parallel work should abort
	// as soon as the first failure is observed. Changes apply only on Sign's success.
	Sign(ctx context.Context, desc *ocmdescriptor.Descriptor) error
}

// ocmSigner signs OCM descriptors against a configurable set of SignatureSpecs.
// Each spec carries its own algorithm, media type, hash, and normalisation algorithm.
// Credentials are resolved per signature through the required credentials.Resolver.
// A missing key for any target signature aborts the entire signing operation.
type ocmSigner struct {
	log       logr.Logger
	resolver  credentials.Resolver
	rsaSigner signing.Signer
	specs     []SignatureSpec
	limiter   Limiter
}

// ocmSignerOption configures an ocmSigner.
type ocmSignerOption func(*ocmSigner)

// withSignerLogger sets the logger for the signer.
func withSignerLogger(log logr.Logger) ocmSignerOption {
	return func(s *ocmSigner) {
		s.log = log
	}
}

// withSignerLimiter installs a Limiter that bounds the number of concurrent
// signing operations. Pass the same Limiter to every Signer and Verifier in the
// process to share the budget. Without this option a NoopLimiter is used —
// signings run unbounded.
func withSignerLimiter(l Limiter) ocmSignerOption {
	return func(s *ocmSigner) {
		if l != nil {
			s.limiter = l
		}
	}
}

func defaultOCMSignerOptions() *ocmSigner {
	return &ocmSigner{
		log:     logr.Discard(),
		limiter: NoopLimiter{},
	}
}

// newOCMSigner creates a new ocmSigner instance.
// A non-nil credentials.Resolver is required. Empty specs are permitted — the
// resulting signer is a no-op (Sign returns nil), mirroring how a Verifier
// treats empty specs. This is the "signing disabled" state.
func newOCMSigner(resolver credentials.Resolver, specs []SignatureSpec, opts ...ocmSignerOption) (*ocmSigner, error) {
	if resolver == nil {
		return nil, fmt.Errorf("create signer: credentials resolver is required")
	}
	if err := specPreFlightSanityCheck(specs); err != nil {
		return nil, fmt.Errorf("create signer: %w", err)
	}
	rsaHandler, err := rsahandler.New(runtime.NewScheme(), false)
	if err != nil {
		return nil, fmt.Errorf("create rsa handler: %w", err)
	}
	s := defaultOCMSignerOptions()
	for _, opt := range opts {
		opt(s)
	}
	names := make([]string, len(specs))
	for i, spec := range specs {
		names[i] = spec.Name
	}
	s.log = s.log.WithValues("signatures", fmt.Sprintf("%v", names))
	s.rsaSigner = rsaHandler
	s.resolver = resolver
	s.specs = specs
	return s, nil
}

func (s *ocmSigner) Sign(ctx context.Context, desc *ocmdescriptor.Descriptor) error {
	// Empty specs is the "signing disabled" state — a no-op, symmetric with the
	// Verifier's empty-specs no-op.
	if len(s.specs) == 0 {
		return nil
	}
	for _, spec := range s.specs {
		if containsSignature(desc.Signatures, spec.Name) {
			return fmt.Errorf("signature with name %q already exists", spec.Name)
		}
	}
	results := make([]ocmdescriptor.Signature, len(s.specs))
	if len(s.specs) == 1 {
		if err := s.sign(ctx, results, 0, desc, s.specs[0]); err != nil {
			return err
		}
	} else {
		// errgroup solves error aggregation and fail-fast: any spec failure cancels
		// gctx, so siblings short-circuit at the next ctx-aware operation. Per-call
		// SetLimit is intentionally absent — concurrency is bounded process-wide by
		// the Limiter installed via withSignerLimiter.
		signerPool, gctx := errgroup.WithContext(ctx)
		for idx, spec := range s.specs {
			signerPool.Go(func() error { return s.sign(gctx, results, idx, desc, spec) })
		}
		if err := signerPool.Wait(); err != nil {
			return fmt.Errorf("signing failed: %w", err)
		}
	}
	desc.Signatures = append(desc.Signatures, results...)
	return nil
}

func (s *ocmSigner) sign(
	ctx context.Context,
	results []ocmdescriptor.Signature,
	idx int,
	desc *ocmdescriptor.Descriptor,
	spec SignatureSpec,
) error {
	release, err := s.limiter.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("sign %q: %w", spec.Name, err)
	}
	defer release()

	rsaConfig, digester := rsaConfigFromSpec(spec), digesterFromSpec(spec)

	dig, err := digester.GenerateDigest(ctx, desc)
	if err != nil {
		return fmt.Errorf("generate digest for %q: %w", spec.Name, err)
	}

	id, err := s.rsaSigner.GetSigningCredentialConsumerIdentity(ctx, spec.Name, *dig, rsaConfig)
	if err != nil {
		return fmt.Errorf("derive consumer identity for %q: %w", spec.Name, err)
	}

	creds, err := s.resolver.Resolve(ctx, id)
	if err != nil {
		return fmt.Errorf("resolve signing credentials for %q: %w", spec.Name, err)
	}

	sigInfo, err := s.rsaSigner.Sign(ctx, *dig, rsaConfig, creds)
	if err != nil {
		return fmt.Errorf("sign %q: %w", spec.Name, err)
	}
	if spec.Issuer != nil {
		sigInfo.Issuer = *spec.Issuer
	}

	results[idx] = ocmdescriptor.Signature{
		Name:      spec.Name,
		Digest:    *dig,
		Signature: sigInfo,
	}
	return nil
}

// rsaConfigFromSpec builds an rsav1alpha1.Config from a SignatureSpec.
// MediaTypePEM maps to SignatureEncodingPolicyPEM; everything else maps to Plain.
func rsaConfigFromSpec(spec SignatureSpec) *rsav1alpha1.Config {
	policy := rsav1alpha1.SignatureEncodingPolicyPlain
	if spec.MediaType == rsav1alpha1.MediaTypePEM {
		policy = rsav1alpha1.SignatureEncodingPolicyPEM
	}
	return &rsav1alpha1.Config{
		Type:                    runtime.NewVersionedType(rsav1alpha1.ConfigType, rsav1alpha1.Version),
		SignatureAlgorithm:      spec.Algorithm,
		SignatureEncodingPolicy: policy,
	}
}

// digesterFromSpec builds a ConfigurableDigester from a SignatureSpec.
func digesterFromSpec(spec SignatureSpec) ConfigurableDigester {
	return NewDigester(WithHashAlgorithm(spec.HashAlgorithm), WithNormalizationAlgorithm(spec.NormalisationAlgorithm))
}

func containsSignature(sigs []ocmdescriptor.Signature, name string) bool {
	for _, s := range sigs {
		if s.Name == name {
			return true
		}
	}
	return false
}

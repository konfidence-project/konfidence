// Package crypto provides RSA signing and verification for Open Component Model (OCM) descriptors.
//
// # Overview
//
// This package implements cryptographic operations for OCM component descriptors, enabling:
//   - RSA-PSS signing of component descriptors with multiple signatures
//   - Signature verification using system trust stores and custom trust anchors
//   - Automatic credential refresh from Kubernetes resources
//
// # Quick Start: Signing
//
// Sign component descriptors using RSASigner with credentials from a Kubernetes Secret:
//
//	import (
//	    "context"
//
//	    "k8s.io/apimachinery/pkg/types"
//	    ctrl "sigs.k8s.io/controller-runtime"
//	    descruntime "ocm.software/open-component-model/bindings/go/descriptor/runtime"
//	    "github.com/konfidence-project/pkg/ocm/crypto"
//	)
//
//	func setupSigning(ctx context.Context, mgr ctrl.Manager, descriptor *descruntime.Descriptor) error {
//	    // Create credential provider that watches a Secret with tls.crt and tls.key
//	    provider := crypto.NewSecretSigningCredentialsProvider(
//	        types.NamespacedName{Name: "signing-key", Namespace: "default"},
//	        crypto.WithNamedSigningProviderLogger(ctrl.Log),
//	    )
//
//	    // Initialize provider - starts watching for Secret updates
//	    if err := provider.SetupWithManager(ctx, mgr); err != nil {
//	        return err
//	    }
//
//	    // Create signer for one or more named signatures
//	    signer, err := crypto.NewRSASigner(
//	        provider,
//	        []string{"my-signature", "backup-signature"},
//	        crypto.WithNamedSignerLogger(ctrl.Log),
//	    )
//	    if err != nil {
//	        return err
//	    }
//
//	    // Sign the descriptor - adds signatures to descriptor.Signatures
//	    return signer.Sign(ctx, descriptor)
//	}
//
// # Quick Start: Verification
//
// Verify component descriptor signatures using RSAVerifier:
//
//	func setupVerification(ctx context.Context, mgr ctrl.Manager, descriptors ...*descruntime.Descriptor) error {
//	    // Optional: add custom trust anchor from ConfigMap
//	    trustProvider := crypto.NewConfigMapTrustAnchorProvider(
//	        types.NamespacedName{Name: "trust-anchor", Namespace: "default"},
//	        crypto.WithNamedTrustAnchorProviderLogger(ctrl.Log),
//	    )
//	    if err := trustProvider.SetupWithManager(ctx, mgr); err != nil {
//	        return err
//	    }
//
//	    // Create verifier - uses system trust store + optional custom anchors
//	    verifier, err := crypto.NewRSAVerifier(
//	        []string{"my-signature"},
//	        crypto.WithVerifierLogger(ctrl.Log),
//	        crypto.WithCredentialProvider(trustProvider),
//	    )
//	    if err != nil {
//	        return err
//	    }
//
//	    // Verify all descriptors - fails if any signature is invalid
//	    return verifier.Verify(ctx, descriptors...)
//	}
//
// # RSASigner
//
// RSASigner signs OCM descriptors using RSA-PSS (RSASSA-PSS) with SHA-256. It supports:
//   - Multiple concurrent signatures with automatic parallelization
//   - Automatic credential refresh from Kubernetes Secrets
//   - Duplicate signature detection
//   - Structured logging with signature tracking
//
// The signer requires a credential provider that supplies tls.crt and tls.key:
//
//	provider := crypto.NewSecretSigningCredentialsProvider(
//	    types.NamespacedName{Name: "my-signing-key", Namespace: "ocm-system"},
//	)
//
//	signer, err := crypto.NewRSASigner(provider, []string{"prod-signature"})
//	if err != nil {
//	    return err
//	}
//
//	// Sign modifies descriptor in place
//	if err := signer.Sign(ctx, descriptor); err != nil {
//	    return err
//	}
//
// Signatures are appended to the descriptor's Signatures slice. Attempting to add a
// duplicate signature name returns an error.
//
// # RSAVerifier
//
// RSAVerifier verifies OCM descriptor signatures using:
//   - System certificate trust store (always enabled)
//   - Optional custom trust anchors from ConfigMaps
//
// Verification checks:
//  1. All required signatures exist in the descriptor
//  2. Digest matches the descriptor content
//  3. Signature is cryptographically valid
//  4. Certificate chain is trusted
//
// Example with system trust store only:
//
//	verifier, err := crypto.NewRSAVerifier([]string{"prod-signature"})
//	if err != nil {
//	    return err
//	}
//
//	if err := verifier.Verify(ctx, descriptor); err != nil {
//	    // Verification failed - signature invalid or not found
//	    return err
//	}
//
// Example with custom trust anchor:
//
//	trustProvider := crypto.NewConfigMapTrustAnchorProvider(
//	    types.NamespacedName{Name: "ca-cert", Namespace: "ocm-system"},
//	)
//	trustProvider.SetupWithManager(ctx, mgr)
//
//	verifier, err := crypto.NewRSAVerifier(
//	    []string{"internal-signature"},
//	    crypto.WithCredentialProvider(trustProvider),
//	)
//
// The verifier automatically handles multiple descriptors concurrently and validates
// all signatures or fails fast on the first error.
//
// # Credential Providers
//
// Credential providers supply keys and certificates for signing/verification:
//
// SecretSigningCredentialsProvider:
//   - Watches a Kubernetes Secret for tls.crt and tls.key
//   - Used with RSASigner for signing operations
//   - Automatically refreshes when Secret updates
//
// ConfigMapTrustAnchorProvider:
//   - Watches a Kubernetes ConfigMap for tls.crt
//   - Used with RSAVerifier to add custom trust anchors
//   - Automatically refreshes when ConfigMap updates
//
// Both providers require SetupWithManager() before use:
//
//	provider := crypto.NewSecretSigningCredentialsProvider(secretRef)
//	if err := provider.SetupWithManager(ctx, mgr); err != nil {
//	    return err
//	}
//
// The provider lifecycle is tied to the context - canceling the context stops updates.
//
// # Disabling Signing/Verification
//
// Use noop implementations to disable operations:
//
//	// Disable signing
//	signer := crypto.NoopSigner{}
//	signer.Sign(ctx, descriptor) // Returns nil, no-op
//
//	// Disable verification
//	verifier := crypto.NoopVerifier{}
//	verifier.Verify(ctx, descriptors...) // Returns nil, no-op
//
// # Error Handling
//
// Common error scenarios:
//
//	// Signing errors
//	- Duplicate signature name already exists
//	- Credentials not available (provider not started or Secret missing)
//	- Invalid private key format
//
//	// Verification errors
//	- Required signature not found in descriptor
//	- Digest mismatch (descriptor was modified)
//	- Invalid signature (cryptographic validation failed)
//	- Untrusted certificate (not in system store or custom anchors)
//
// All errors are wrapped with context for debugging.
//
// # Best Practices
//
//  1. Initialize providers early: Call SetupWithManager during controller setup,
//     before creating signers/verifiers.
//
//  2. Use named loggers: Enable WithNamedSignerLogger/WithNamedVerifierLogger
//     for clear log attribution in multi-component systems.
//
//  3. Share providers: One provider can serve multiple signers/verifiers,
//     reducing Kubernetes API overhead.
package crypto

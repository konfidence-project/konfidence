package ocm

import (
	"context"

	ocmcredentials "ocm.software/open-component-model/bindings/go/credentials"
	credcfgruntime "ocm.software/open-component-model/bindings/go/credentials/spec/config/runtime"
	ocmdescriptor "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	rsav1alpha1 "ocm.software/open-component-model/bindings/go/rsa/signing/v1alpha1"
	rsacredentials "ocm.software/open-component-model/bindings/go/rsa/spec/credentials"
	rsacredv1 "ocm.software/open-component-model/bindings/go/rsa/spec/credentials/v1"
	rsaidentityv1 "ocm.software/open-component-model/bindings/go/rsa/spec/identity/v1"
	ocmruntime "ocm.software/open-component-model/bindings/go/runtime"

	. "github.com/onsi/gomega" //nolint:staticcheck

	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
)

// SignDescriptor signs desc in-place for every (signatureName, pair) binding using
// DefaultSignatureSpec defaults. Builds a transient in-memory resolver directly
// from the RSA key material — no Kubernetes API, no fake client, no filesystem.
//
// Call this before pushing to Zot to simulate out-of-band artifact pre-signing.
// Fails the current Gomega test on any error, reporting at the caller's location.
func SignDescriptor(ctx context.Context, desc *ocmdescriptor.Descriptor, bindings ...SignatureBinding) {
	if len(bindings) == 0 {
		return
	}

	resolver := resolverFromBindings(ctx, bindings)

	specs := make([]crypto.SignatureSpec, len(bindings))
	for i, b := range bindings {
		specs[i] = crypto.DefaultSignatureSpec(b.SignatureName, nil)
	}

	signer, err := crypto.NewOCMSigner(resolver, specs)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "create signer for SignDescriptor")
	ExpectWithOffset(1, signer.Sign(ctx, desc)).NotTo(HaveOccurred(), "sign descriptor")
}

// resolverFromBindings builds an ocmcredentials.Resolver directly from RSA key material
// without touching Kubernetes or the filesystem.
func resolverFromBindings(ctx context.Context, bindings []SignatureBinding) ocmcredentials.Resolver {
	consumers := make([]credcfgruntime.Consumer, 0, len(bindings))
	for _, b := range bindings {
		identity := ocmruntime.Identity{
			rsaidentityv1.IdentityAttributeSignature: b.SignatureName,
			rsaidentityv1.IdentityAttributeAlgorithm: string(rsav1alpha1.AlgorithmRSASSAPSS),
		}
		identity.SetType(rsaidentityv1.V1Alpha1Type)

		consumers = append(consumers, credcfgruntime.Consumer{
			Identities: []ocmruntime.Identity{identity},
			Credentials: []ocmruntime.Typed{&rsacredv1.RSACredentials{
				Type:          rsacredv1.VersionedType,
				PrivateKeyPEM: string(b.Pair.PrivateKeyPEM),
				PublicKeyPEM:  string(b.Pair.CertificatePEM),
			}},
		})
	}

	resolver, err := ocmcredentials.ToGraph(ctx, &credcfgruntime.Config{Consumers: consumers}, ocmcredentials.Options{
		CredentialTypeSchemeProvider: rsaCredTypeScheme(),
	})
	ExpectWithOffset(2, err).NotTo(HaveOccurred(), "build credential graph for SignDescriptor")
	return resolver
}

// rsaCredTypeScheme returns a CredentialTypeSchemeProvider with RSACredentials/v1 registered
// so ToGraph treats RSA credentials as first-class typed values rather than routing them through
// the credential plugin path (which would nil-panic without a plugin provider).
func rsaCredTypeScheme() ocmcredentials.CredentialTypeSchemeProvider {
	scheme := ocmruntime.NewScheme()
	rsacredentials.MustRegisterCredentialType(scheme)
	return &rsaCredTypeSchemeProvider{scheme: scheme}
}

type rsaCredTypeSchemeProvider struct{ scheme *ocmruntime.Scheme }

func (r *rsaCredTypeSchemeProvider) GetCredentialTypeScheme() *ocmruntime.Scheme { return r.scheme }

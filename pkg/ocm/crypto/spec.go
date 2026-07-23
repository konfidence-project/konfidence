package crypto

import (
	"crypto"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	norm "ocm.software/open-component-model/bindings/go/descriptor/normalisation/json/v4alpha1"
	rsav1alpha1 "ocm.software/open-component-model/bindings/go/rsa/signing/v1alpha1"
)

// SignatureSpec pins the complete cryptographic policy for one named signature.
// Construct via NewSignatureSpec or DefaultSignatureSpec.
type SignatureSpec struct {
	// Name is the OCM signature name matched against the descriptor.
	Name string
	// Algorithm is the RSA signing algorithm.
	Algorithm rsav1alpha1.SignatureAlgorithm
	// MediaType is the signature encoding format.
	MediaType string
	// HashAlgorithm is the hash function used to produce the content digest.
	HashAlgorithm string
	// NormalisationAlgorithm is the serialization normalization algorithm.
	NormalisationAlgorithm string
	// Issuer is the expected certificate issuer DN.
	Issuer *string
}

// NewSignatureSpec constructs a SignatureSpec from the given parameters.
func NewSignatureSpec(
	name string,
	algorithm rsav1alpha1.SignatureAlgorithm,
	mediaType string,
	hashAlgorithm string,
	normalisationAlgorithm string,
	issuer *string,
) SignatureSpec {
	return SignatureSpec{
		Name:                   name,
		Algorithm:              algorithm,
		MediaType:              mediaType,
		HashAlgorithm:          hashAlgorithm,
		NormalisationAlgorithm: normalisationAlgorithm,
		Issuer:                 issuer,
	}
}

// DefaultSignatureSpec returns a SignatureSpec with secure defaults:
//   - Algorithm:              RSASSA-PSS
//   - MediaType:              application/x-pem-file (PEM)
//   - HashAlgorithm:          SHA-256
//   - NormalisationAlgorithm: jsonNormalisation/v4alpha1
func DefaultSignatureSpec(name string, issuer *string) SignatureSpec {
	return NewSignatureSpec(
		name,
		rsav1alpha1.AlgorithmRSASSAPSS,
		rsav1alpha1.MediaTypePEM,
		crypto.SHA256.String(),
		norm.Algorithm,
		issuer,
	)
}

// SpecsFromVerify converts a *konfidence.Verify into a []SignatureSpec, one per signature entry.
// Returns an empty slice if v is nil — the builder will produce a NoopVerifier.
func SpecsFromVerify(v *konfidence.Verify) []SignatureSpec {
	if v == nil {
		return []SignatureSpec{}
	}
	specs := make([]SignatureSpec, len(v.Signatures))
	for i, sig := range v.Signatures {
		specs[i] = NewSignatureSpecFromV1alpha1(sig)
	}
	return specs
}

// SpecsFromSign converts a *konfidence.Sign into a []SignatureSpec, one per signature entry.
// Returns an empty slice if s is nil — the builder will produce a NoopSigner.
func SpecsFromSign(s *konfidence.Sign) []SignatureSpec {
	if s == nil {
		return []SignatureSpec{}
	}
	specs := make([]SignatureSpec, len(s.Signatures))
	for i, sig := range s.Signatures {
		specs[i] = NewSignatureSpecFromV1alpha1(sig)
	}
	return specs
}

// NewSignatureSpecFromV1alpha1 constructs a SignatureSpec from a konfidence.Signature.
// It takes a DefaultSignatureSpec as a base and overrides any corresponding fields that are set on the input konfidence.Signature.
func NewSignatureSpecFromV1alpha1(signature konfidence.Signature) SignatureSpec {
	spec := DefaultSignatureSpec(signature.Name, signature.Issuer)
	if signature.Algorithm != nil {
		spec.Algorithm = rsav1alpha1.SignatureAlgorithm(*signature.Algorithm)
	}
	if signature.SignatureMediaType != nil {
		spec.MediaType = *signature.SignatureMediaType
	}
	if signature.HashAlgorithm != nil {
		spec.HashAlgorithm = *signature.HashAlgorithm
	}
	if signature.NormalisationAlgorithm != nil {
		spec.NormalisationAlgorithm = *signature.NormalisationAlgorithm
	}
	return spec
}

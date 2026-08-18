package v1alpha1

// Signature pins parameters of one named signature on a component
// descriptor. Used both for verification (matched against the fetched
// descriptor) and for signing (overrides defaults of the emitted
// signature).
//
// +kubebuilder:validation:XValidation:rule="!has(self.algorithm) || self.algorithm in ['RSASSA-PSS', 'RSASSA-PKCS1-V1_5']",message="algorithm must be one of: RSASSA-PSS, RSASSA-PKCS1-V1_5"
// +kubebuilder:validation:XValidation:rule="!has(self.signatureMediaType) || self.signatureMediaType in ['application/x-pem-file', 'application/vnd.ocm.signature.rsa.pss', 'application/vnd.ocm.signature.rsa']",message="signatureMediaType must be one of: application/x-pem-file, application/vnd.ocm.signature.rsa.pss, application/vnd.ocm.signature.rsa"
// +kubebuilder:validation:XValidation:rule="!has(self.hashAlgorithm) || self.hashAlgorithm in ['SHA-256', 'SHA-512']",message="hashAlgorithm must be one of: SHA-256, SHA-512"
// +kubebuilder:validation:XValidation:rule="!has(self.normalisationAlgorithm) || self.normalisationAlgorithm in ['jsonNormalisation/v4alpha1']",message="normalisationAlgorithm must be: jsonNormalisation/v4alpha1"
// +kubebuilder:validation:XValidation:rule="!has(self.issuer) || size(self.issuer) > 0",message="issuer must be non-empty when set"
type Signature struct {
	// Name is the unique identifier for this signature.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Algorithm specifies the RSA signing algorithm.
	// When omitted, RSASSA-PSS is used.
	// Valid values: RSASSA-PSS, RSASSA-PKCS1-V1_5.
	// +optional
	Algorithm *string `json:"algorithm,omitempty"`

	// SignatureMediaType specifies the encoding format for the signature bytes.
	// When omitted, application/x-pem-file (PEM) is used.
	// Valid values: application/x-pem-file, application/vnd.ocm.signature.rsa.pss,
	// application/vnd.ocm.signature.rsa.
	// +optional
	SignatureMediaType *string `json:"signatureMediaType,omitempty"`

	// HashAlgorithm specifies the digest algorithm used when hashing the component descriptor.
	// When omitted, SHA-256 is used.
	// Valid values: SHA-256, SHA-512.
	// +optional
	HashAlgorithm *string `json:"hashAlgorithm,omitempty"`

	// NormalisationAlgorithm specifies the normalisation scheme applied to the descriptor
	// before hashing.
	// When omitted, jsonNormalisation/v4alpha1 is used.
	// Valid values: jsonNormalisation/v4alpha1.
	// +optional
	NormalisationAlgorithm *string `json:"normalisationAlgorithm,omitempty"`

	// Issuer pins the expected certificate issuer DN for PEM-encoded signatures.
	// On the sign path the value is stamped into the descriptor alongside the signature,
	// so it is enforced automatically on the verify path even without an explicit pin here.
	// On the verify path, when set, this value overrides whatever the descriptor stored and
	// the handler rejects any signature whose leaf certificate issuer DN does not match.
	// When omitted on both paths the issuer field stays empty and no DN check is performed.
	// Must be non-empty when present.
	// +optional
	Issuer *string `json:"issuer,omitempty"`
}

// Verify lists candidate signatures evaluated against every fetched
// descriptor. Absence on a spec disables verification.
type Verify struct {
	// +kubebuilder:validation:MinItems=1
	Signatures []Signature `json:"signatures"`
}

// Sign lists signatures the controller produces on every descriptor it
// writes. Absence on a spec disables signing.
type Sign struct {
	// +kubebuilder:validation:MinItems=1
	Signatures []Signature `json:"signatures"`
}

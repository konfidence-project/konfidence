package crypto

import (
	"crypto"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	norm "ocm.software/open-component-model/bindings/go/descriptor/normalisation/json/v4alpha1"
	rsav1alpha1 "ocm.software/open-component-model/bindings/go/rsa/signing/v1alpha1"
)

var _ = Describe("SpecsFromVerify", func() {
	DescribeTable("converts Verify.Signatures into []SignatureSpec",
		func(verify *v1alpha1.Verify, expected []SignatureSpec) {
			Expect(SpecsFromVerify(verify)).To(Equal(expected))
		},
		Entry("returns []SignatureSpec{} in case input is nil", nil, []SignatureSpec{}),
		Entry("empty signatures", &v1alpha1.Verify{}, []SignatureSpec{}),
		Entry("single signature with defaults",
			&v1alpha1.Verify{Signatures: []v1alpha1.Signature{{Name: "sig-a"}}},
			[]SignatureSpec{
				{
					Name:                   "sig-a",
					Algorithm:              rsav1alpha1.AlgorithmRSASSAPSS,
					MediaType:              rsav1alpha1.MediaTypePEM,
					HashAlgorithm:          crypto.SHA256.String(),
					NormalisationAlgorithm: norm.Algorithm,
					Issuer:                 nil,
				},
			},
		),
		Entry("single signature with all fields overridden",
			&v1alpha1.Verify{Signatures: []v1alpha1.Signature{{
				Name:                   "sig-b",
				Algorithm:              new("RSA_FANCY"),
				SignatureMediaType:     new("myMediaType"),
				HashAlgorithm:          new("SHA256"),
				NormalisationAlgorithm: new("my_norm"),
				Issuer:                 new("my_issuer"),
			}}},
			[]SignatureSpec{{
				Name:                   "sig-b",
				Algorithm:              "RSA_FANCY",
				MediaType:              "myMediaType",
				HashAlgorithm:          "SHA256",
				NormalisationAlgorithm: "my_norm",
				Issuer:                 new("my_issuer"),
			}},
		),
		Entry("multiple signatures",
			&v1alpha1.Verify{Signatures: []v1alpha1.Signature{
				{Name: "sig-1"},
				{Name: "sig-2", Issuer: new("CN=ca2")},
			}},
			[]SignatureSpec{
				DefaultSignatureSpec("sig-1", nil),
				DefaultSignatureSpec("sig-2", new("CN=ca2")),
			},
		),
	)
})

var _ = Describe("SpecsFromSign", func() {
	DescribeTable("converts Sign.Signatures into []SignatureSpec",
		func(sign *v1alpha1.Sign, expected []SignatureSpec) {
			Expect(SpecsFromSign(sign)).To(Equal(expected))
		},
		Entry("returns []SignatureSpec{} in case input is nil", nil, []SignatureSpec{}),
		Entry("empty signatures", &v1alpha1.Sign{}, []SignatureSpec{}),
		Entry("single signature with defaults",
			&v1alpha1.Sign{Signatures: []v1alpha1.Signature{{Name: "sign-a"}}},
			[]SignatureSpec{DefaultSignatureSpec("sign-a", nil)},
		),
		Entry("single signature with issuer",
			&v1alpha1.Sign{Signatures: []v1alpha1.Signature{{
				Name:   "sign-b",
				Issuer: new("CN=ca2"),
			}}},
			[]SignatureSpec{DefaultSignatureSpec("sign-b", new("CN=ca2"))},
		),
	)
})

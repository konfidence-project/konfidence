package controller

import (
	"context"
	"encoding/json"
	"time"

	galaxy "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	"github.com/konfidence-project/konfidence/pkg/jsonschema"
	"github.com/konfidence-project/konfidence/pkg/testutil/ocm"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"ocm.software/open-component-model/bindings/go/oci/compref"
)

// createReference creates a reference and fails the test in case of errors.
func createReference(component string) compref.Ref {
	return ocm.ParseRef(registryEndpoint, component)
}

// createVectorTemplateCR creates a VectorTemplate CR with OCI credentials pre-wired.
//
//nolint:unparam // namespace is the same in every call, keep as param for consistency
func createVectorTemplateCR(
	ctx context.Context,
	name, namespace string,
	artifacts []compref.Ref,
	vector compref.Ref,
	base *compref.Ref,
	vectorConfig *galaxy.VectorConfig) *galaxy.VectorTemplate {
	return createPKIVectorTemplateCR(ctx, name, namespace, artifacts, vector, base,
		pkiVectorTemplateOptions{credSecretNames: credSecretNames, vectorConfig: vectorConfig},
	)
}

func getVectorConfigurationContent(vectorConfig galaxy.VectorConfig) ([]byte, error) {
	var features json.RawMessage
	if vectorConfig.Features != nil {
		features = json.RawMessage(vectorConfig.Features.Raw)
	}
	var authored json.RawMessage
	if vectorConfig.Authored != nil {
		authored = json.RawMessage(vectorConfig.Authored.Raw)
	}
	content, err := json.Marshal(jsonschema.NewVectorConfigurationV1(features, authored))
	if err != nil {
		return nil, err
	}
	return content, nil
}

// pkiVectorTemplateOptions holds optional fields for createPKIVectorTemplateCR.
type pkiVectorTemplateOptions struct {
	credSecretNames []string
	vectorConfig    *galaxy.VectorConfig
	signVector      *galaxy.Sign
	verifyVector    *galaxy.Verify
	verifyArtifacts *galaxy.Verify
}

// createPKIVectorTemplateCR creates a VectorTemplate CR with optional credential refs,
// sign, verify specs, and vectorConfig wired in. Pass nil slices/specs to omit the corresponding field.
//
//nolint:unparam // namespace is the same in every call, keep as param for consistency
func createPKIVectorTemplateCR(
	ctx context.Context,
	name, namespace string,
	artifacts []compref.Ref,
	vector compref.Ref,
	base *compref.Ref,
	opts pkiVectorTemplateOptions,
) *galaxy.VectorTemplate {
	components := make([]galaxy.Component, 0, len(artifacts))
	for _, artifact := range artifacts {
		components = append(components, galaxy.Component{Name: artifact.String()})
	}
	var baseRef *string
	if base != nil {
		baseRef = new(base.String())
	}

	var creds *galaxy.Credentials
	if len(opts.credSecretNames) > 0 {
		refs := make([]galaxy.CredentialRef, len(opts.credSecretNames))
		for i, n := range opts.credSecretNames {
			refs[i] = galaxy.CredentialRef{Name: n}
		}
		creds = &galaxy.Credentials{OCM: &galaxy.OCMCredentials{Refs: refs}}
	}

	vt := &galaxy.VectorTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: galaxy.VectorTemplateSpec{
			ReconcileInterval: &metav1.Duration{Duration: time.Hour},
			UploadTarget:      vector.String(),
			Base:              baseRef,
			Components:        components,
			Credentials:       creds,
			VectorConfig:      opts.vectorConfig,
			SignVector:        opts.signVector,
			VerifyVector:      opts.verifyVector,
			VerifyArtifacts:   opts.verifyArtifacts,
		},
	}
	Expect(k8sClient.Create(ctx, vt)).To(Succeed())
	return vt
}

// signSpec returns a *galaxy.Sign with a single signature using the given name and all defaults.
func signSpec(sigName string) *galaxy.Sign {
	return &galaxy.Sign{Signatures: []galaxy.Signature{{Name: sigName}}}
}

// verifySpec returns a *galaxy.Verify with a single signature using the given name and all defaults.
func verifySpec(sigName string) *galaxy.Verify {
	return &galaxy.Verify{Signatures: []galaxy.Signature{{Name: sigName}}}
}

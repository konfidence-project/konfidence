package controller

import (
	"context"
	"encoding/json"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/pkg/jsonschema"
	"github.com/konfidence-project/konfidence/pkg/testutil/ocm"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// createReference creates a reference and fails the test in case of errors.
func createReference(component string) compref.Ref {
	return ocm.ParseRef(registryEndpoint, component)
}

// createBareReference creates a version-less reference (no tag) for use as a VectorTemplate
// uploadTarget, and fails the test in case of errors.
func createBareReference(component string) compref.Ref {
	return ocm.ParseBareRef(registryEndpoint, component)
}

// updateVectorConfig sets the VectorTemplate's spec.vectorConfig and persists it. Because it
// changes the spec, it bumps the generation and triggers a reconcile via the For predicate.
// Pass nil to remove the vector configuration.
func updateVectorConfig(vt *konfidence.VectorTemplate, cfg *konfidence.VectorConfig) {
	GinkgoHelper()
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(vt), vt)).To(Succeed())
		vt.Spec.VectorConfig = cfg
		g.Expect(k8sClient.Update(ctx, vt)).To(Succeed())
	}, timeout, interval).Should(Succeed())
}

// nudgeReconcile forces the controller to reconcile the VectorTemplate again by bumping a
// spec field (reconcileInterval), which changes the object generation and therefore passes
// the GenerationChangedPredicate on the For watch. Used to re-drive assembly after the OCI
// state (e.g. a component alias) has changed, without altering the desired component set.
func nudgeReconcile(vt *konfidence.VectorTemplate) {
	GinkgoHelper()
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(vt), vt)).To(Succeed())
		cur := time.Hour
		if vt.Spec.ReconcileInterval != nil {
			cur = vt.Spec.ReconcileInterval.Duration
		}
		// Toggle between two long intervals so the generation changes but reconcile
		// cadence stays effectively disabled (tests drive reconciles explicitly).
		next := time.Hour
		if cur == time.Hour {
			next = 2 * time.Hour
		}
		vt.Spec.ReconcileInterval = &metav1.Duration{Duration: next}
		g.Expect(k8sClient.Update(ctx, vt)).To(Succeed())
	}, timeout, interval).Should(Succeed())
}

// createVectorTemplateCR creates a VectorTemplate CR with OCI credentials pre-wired.
//
//nolint:unparam // namespace is the same in every call, keep as param for consistency
func createVectorTemplateCR(
	ctx context.Context,
	name, namespace string,
	artifacts []compref.Ref,
	vector compref.Ref,
	baseName string,
	vectorConfig *konfidence.VectorConfig) *konfidence.VectorTemplate {
	return createPKIVectorTemplateCR(ctx, name, namespace, artifacts, vector, baseName,
		pkiVectorTemplateOptions{credSecretNames: credSecretNames, vectorConfig: vectorConfig},
	)
}

func getVectorConfigurationContent(vectorConfig konfidence.VectorConfig) ([]byte, error) {
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
	vectorConfig    *konfidence.VectorConfig
	signVector      *konfidence.Sign
	verifyVector    *konfidence.Verify
	verifyArtifacts *konfidence.Verify
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
	baseName string,
	opts pkiVectorTemplateOptions,
) *konfidence.VectorTemplate {
	components := make([]konfidence.Component, 0, len(artifacts))
	for _, artifact := range artifacts {
		components = append(components, konfidence.Component{Name: artifact.String()})
	}
	var baseRef *konfidence.VectorTemplateReference
	if baseName != "" {
		baseRef = &konfidence.VectorTemplateReference{
			Kind: konfidence.VectorTemplateKind,
			Name: baseName,
		}
	}

	var creds *konfidence.Credentials
	if len(opts.credSecretNames) > 0 {
		refs := make([]konfidence.CredentialRef, len(opts.credSecretNames))
		for i, n := range opts.credSecretNames {
			refs[i] = konfidence.CredentialRef{Name: n}
		}
		creds = &konfidence.Credentials{OCM: &konfidence.OCMCredentials{Refs: refs}}
	}

	vt := &konfidence.VectorTemplate{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: konfidence.VectorTemplateSpec{
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

// signSpec returns a *konfidence.Sign with a single signature using the given name and all defaults.
func signSpec(sigName string) *konfidence.Sign {
	return &konfidence.Sign{Signatures: []konfidence.Signature{{Name: sigName}}}
}

// verifySpec returns a *konfidence.Verify with a single signature using the given name and all defaults.
func verifySpec(sigName string) *konfidence.Verify {
	return &konfidence.Verify{Signatures: []konfidence.Signature{{Name: sigName}}}
}

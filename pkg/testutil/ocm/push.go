package ocm

import (
	"bytes"
	"context"
	gocrypto "crypto"
	"fmt"
	"log/slog"

	konfcompref "github.com/konfidence-project/konfidence/pkg/ocm/compref"
	pkgocm "github.com/konfidence-project/konfidence/pkg/ocm/repository"
	"github.com/onsi/gomega"
	"github.com/opencontainers/go-digest"
	"ocm.software/open-component-model/bindings/go/blob/inmemory"
	norm "ocm.software/open-component-model/bindings/go/descriptor/normalisation/json/v4alpha1"
	ocmdescriptor "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	v2 "ocm.software/open-component-model/bindings/go/descriptor/v2"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	"ocm.software/open-component-model/bindings/go/signing"
)

// buildReference constructs a vector reference entry with the given digest.
func buildReference(i int, artifact compref.Ref, digestValue string) ocmdescriptor.Reference {
	return ocmdescriptor.Reference{
		ElementMeta: ocmdescriptor.ElementMeta{
			ObjectMeta: ocmdescriptor.ObjectMeta{
				Name:    fmt.Sprintf("ref-%d", i),
				Version: artifact.Version,
			},
		},
		Component: artifact.Component,
		Digest: ocmdescriptor.Digest{
			HashAlgorithm:          gocrypto.SHA256.String(),
			NormalisationAlgorithm: norm.Algorithm,
			Value:                  digestValue,
		},
	}
}

// computeDigest fetches a descriptor and computes its normalised SHA-256 digest.
// Returns the zero digest string if the artifact cannot be fetched (e.g. not yet pushed).
func computeDigest(ctx context.Context, client pkgocm.Client, ref compref.Ref) string {
	desc, err := client.Get(ctx, ref)
	if err != nil {
		return "0000000000000000000000000000000000000000000000000000000000000000"
	}
	dig, err := signing.GenerateDigest(ctx, &desc, slog.Default(), norm.Algorithm, gocrypto.SHA256.String())
	if err != nil {
		return "0000000000000000000000000000000000000000000000000000000000000000"
	}
	return dig.Value
}

func buildComponentDescriptor(ref compref.Ref) ocmdescriptor.Descriptor {
	return ocmdescriptor.Descriptor{
		Meta: ocmdescriptor.Meta{Version: "v2"},
		Component: ocmdescriptor.Component{
			ComponentMeta: ocmdescriptor.ComponentMeta{
				ObjectMeta: ocmdescriptor.ObjectMeta{
					Name:    ref.Component,
					Version: ref.Version,
				},
			},
			Provider: ocmdescriptor.Provider{Name: "test"},
		},
	}
}

// SampleVectorConfig returns a representative vector config JSON payload for use in tests
// that need a non-nil config but don't care about the specific content.
func SampleVectorConfig() []byte {
	return []byte(`{"apiVersion":"konfidence.cloud/v1alpha1","kind":"VectorConfiguration","features":{"env":"test"},"authored":{"owner":"sample"}}`)
}

// addVectorConfigResource embeds vectorConfig as a local JSON resource in the descriptor.
func addVectorConfigResource(ctx context.Context, client pkgocm.Client, vector compref.Ref, descriptor *ocmdescriptor.Descriptor, vectorConfig []byte) {
	resource := &ocmdescriptor.Resource{
		Relation: ocmdescriptor.LocalRelation,
		ElementMeta: ocmdescriptor.ElementMeta{
			ObjectMeta: ocmdescriptor.ObjectMeta{
				Name:    "cloud-konfidence-vector-config",
				Version: "1.0.0",
			},
		},
		Type: "json",
		Access: &v2.LocalBlob{
			LocalReference: digest.FromBytes(vectorConfig).String(),
			MediaType:      "application/json",
		},
	}
	content := inmemory.New(bytes.NewReader(vectorConfig))
	updatedResource, err := client.AddLocalResource(ctx, vector.Repository, *descriptor, *resource, content)
	gomega.ExpectWithOffset(2, err).NotTo(gomega.HaveOccurred(), "failed to add vector config resource to %s", vector)
	descriptor.Component.Resources = append(descriptor.Component.Resources, *updatedResource)
}

func buildVectorDescriptor(ctx context.Context, client pkgocm.Client, vector compref.Ref, artifacts []compref.Ref) ocmdescriptor.Descriptor {
	references := make([]ocmdescriptor.Reference, 0, len(artifacts))
	for i, artifact := range artifacts {
		references = append(references, buildReference(i, artifact, computeDigest(ctx, client, artifact)))
	}
	return ocmdescriptor.Descriptor{
		Meta: ocmdescriptor.Meta{Version: "v2"},
		Component: ocmdescriptor.Component{
			ComponentMeta: ocmdescriptor.ComponentMeta{
				ObjectMeta: ocmdescriptor.ObjectMeta{
					Name:    vector.Component,
					Version: vector.Version,
				},
			},
			Provider:   ocmdescriptor.Provider{Name: "konfidence"},
			References: references,
		},
	}
}

// PushSignedComponent pushes a minimal OCM component descriptor pre-signed with the
// given bindings into the OCI registry identified by ref. If alias is non-nil the
// component is additionally tagged with that alias.
//
// Signing is performed out-of-band before the push — no controller involvement.
// Fails the current Gomega test on any error, reporting at the caller's location.
func PushSignedComponent(ctx context.Context, client pkgocm.Client, ref compref.Ref, alias *string, bindings ...SignatureBinding) {
	descriptor := buildComponentDescriptor(ref)
	SignDescriptor(ctx, &descriptor, bindings...)
	gomega.ExpectWithOffset(1,
		client.Save(ctx, ref.Repository, descriptor)).
		NotTo(gomega.HaveOccurred(), "failed to push signed component %s", ref)
	if alias != nil {
		gomega.ExpectWithOffset(1,
			client.AddAlias(ctx, ref, *alias)).
			NotTo(gomega.HaveOccurred(), "failed to add alias %s for signed component %s", *alias, ref)
	}
}

// PushSignedVector pushes a vector descriptor (an OCM component with references to
// artifact components) pre-signed with the given bindings, tagged with alias.
// If vectorConfig is non-nil it is embedded as a local JSON resource before signing.
//
// Signing is performed out-of-band before the push — no controller involvement.
// Fails the current Gomega test on any error, reporting at the caller's location.
func PushSignedVector(
	ctx context.Context, client pkgocm.Client, vector compref.Ref,
	artifacts []compref.Ref, alias string, vectorConfig []byte, bindings ...SignatureBinding,
) {
	descriptor := buildVectorDescriptor(ctx, client, vector, artifacts)
	if vectorConfig != nil {
		addVectorConfigResource(ctx, client, vector, &descriptor, vectorConfig)
	}
	SignDescriptor(ctx, &descriptor, bindings...)
	gomega.ExpectWithOffset(1,
		client.Save(ctx, vector.Repository, descriptor)).
		NotTo(gomega.HaveOccurred(), "failed to push signed vector %s", vector)
	gomega.ExpectWithOffset(1,
		client.AddAlias(ctx, vector, alias)).
		NotTo(gomega.HaveOccurred(), "failed to add alias %s for signed vector %s", alias, vector)
}

// ParseRef parses a full component reference string of the form
// "http://<registry>//<component>:<version>" and fails the current Gomega test
// if parsing fails, reporting the failure at the caller's location (offset 1).
func ParseRef(registryEndpoint, component string) compref.Ref {
	ref, err := konfcompref.Parse(fmt.Sprintf("http://%s//%s", registryEndpoint, component))
	gomega.ExpectWithOffset(1, err).
		NotTo(gomega.HaveOccurred(), "failed to parse reference for component %s", component)
	return *ref
}

// ParseBareRef parses a version-less component reference of the form
// "http://<registry>//<component>" (no version/alias tag) and fails the current Gomega
// test if parsing fails. Use this for a VectorTemplate uploadTarget, which must not carry
// a version - the controller assigns the concrete version itself.
func ParseBareRef(registryEndpoint, component string) compref.Ref {
	ref, err := konfcompref.Parse(
		fmt.Sprintf("http://%s//%s", registryEndpoint, component),
		konfcompref.WithVersionValidation(konfcompref.VersionValidationNoVersion))
	gomega.ExpectWithOffset(1, err).
		NotTo(gomega.HaveOccurred(), "failed to parse bare reference for component %s", component)
	return *ref
}

// PushComponent pushes a minimal OCM component descriptor into the OCI registry
// identified by ref. If alias is non-nil the component is additionally tagged
// with that alias via AddAlias.
//
// The function fails the current Gomega test on any error, reporting the failure
// at the caller's location (offset 1).
func PushComponent(ctx context.Context, client pkgocm.Client, ref compref.Ref, alias *string) {
	descriptor := buildComponentDescriptor(ref)
	gomega.ExpectWithOffset(1,
		client.Save(ctx, ref.Repository, descriptor)).
		NotTo(gomega.HaveOccurred(), "failed to push component %s", ref)
	if alias != nil {
		gomega.ExpectWithOffset(1,
			client.AddAlias(ctx, ref, *alias)).
			NotTo(gomega.HaveOccurred(), "failed to add alias %s for component %s", *alias, ref)
	}
}

// PushVector pushes a vector descriptor (an OCM component with references to
// artifact components) into the OCI registry at its concrete version. If alias is
// non-empty the vector is additionally tagged with it via AddAlias; pass "" to push
// the vector without moving any alias (the model used since ADR-0032, where vectors
// are referenced by concrete version rather than an alias tag).
// If vectorConfig is non-nil it is embedded as a local JSON resource named
// "cloud-konfidence-vector-config" inside the vector descriptor.
//
// The function fails the current Gomega test on any error, reporting the failure
// at the caller's location (offset 1).
//
// TODO: drop the alias parameter once all controllers stop using vector aliases.
func PushVector(ctx context.Context, client pkgocm.Client, vector compref.Ref, artifacts []compref.Ref, alias string, vectorConfig []byte) {
	descriptor := buildVectorDescriptor(ctx, client, vector, artifacts)

	if vectorConfig != nil {
		addVectorConfigResource(ctx, client, vector, &descriptor, vectorConfig)
	}

	gomega.ExpectWithOffset(1,
		client.Save(ctx, vector.Repository, descriptor)).
		NotTo(gomega.HaveOccurred(), "failed to push vector %s", vector)
	if alias == "" {
		return
	}
	gomega.ExpectWithOffset(1,
		client.AddAlias(ctx, vector, alias)).
		NotTo(gomega.HaveOccurred(), "failed to add alias %s for vector %s", alias, vector)
}

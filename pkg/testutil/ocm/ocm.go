package ocm

import (
	"bytes"
	"context"
	"fmt"

	konfcompref "github.com/konfidence-project/konfidence/pkg/ocm/compref"
	pkgocm "github.com/konfidence-project/konfidence/pkg/ocm/repository"
	"github.com/onsi/gomega"
	"github.com/opencontainers/go-digest"
	"ocm.software/open-component-model/bindings/go/blob/inmemory"
	ocmdescriptor "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	v2 "ocm.software/open-component-model/bindings/go/descriptor/v2"
	"ocm.software/open-component-model/bindings/go/oci/compref"
)

// ParseRef parses a full component reference string of the form
// "http://<registry>//<component>:<version>" and fails the current Gomega test
// if parsing fails, reporting the failure at the caller's location (offset 1).
func ParseRef(registryEndpoint, component string) compref.Ref {
	ref, err := konfcompref.Parse(fmt.Sprintf("http://%s//%s", registryEndpoint, component))
	gomega.ExpectWithOffset(1, err).
		NotTo(gomega.HaveOccurred(), "failed to parse reference for component %s", component)
	return *ref
}

// PushComponent pushes a minimal OCM component descriptor into the OCI registry
// identified by ref. If alias is non-nil the component is additionally tagged
// with that alias via AddAlias.
//
// The function fails the current Gomega test on any error, reporting the failure
// at the caller's location (offset 1).
func PushComponent(ctx context.Context, client pkgocm.Client, ref compref.Ref, alias *string) {
	descriptor := ocmdescriptor.Descriptor{
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
// artifact components) into the OCI registry and tags it with the given alias.
//
// The generated references are named "ref-0", "ref-1", … in the order supplied.
//
// The function fails the current Gomega test on any error, reporting the failure
// at the caller's location (offset 1).
func PushVector(ctx context.Context, client pkgocm.Client, vector compref.Ref, artifacts []compref.Ref, alias string, vectorConfig []byte) {
	references := make([]ocmdescriptor.Reference, 0, len(artifacts))
	for i, artifact := range artifacts {
		references = append(references, ocmdescriptor.Reference{
			ElementMeta: ocmdescriptor.ElementMeta{
				ObjectMeta: ocmdescriptor.ObjectMeta{
					Name:    fmt.Sprintf("ref-%d", i),
					Version: artifact.Version,
				},
			},
			Component: artifact.Component,
		})
	}

	vectorDescriptor := ocmdescriptor.Descriptor{
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

	if vectorConfig != nil {
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
		updatedResource, err := client.AddLocalResource(ctx, vector.Repository, vectorDescriptor, *resource, content)
		gomega.Expect(err).NotTo(gomega.HaveOccurred())
		vectorDescriptor.Component.Resources = append(vectorDescriptor.Component.Resources, *updatedResource)
	}

	gomega.ExpectWithOffset(1,
		client.Save(ctx, vector.Repository, vectorDescriptor)).
		NotTo(gomega.HaveOccurred(), "failed to push vector %s", vector)
	gomega.ExpectWithOffset(1,
		client.AddAlias(ctx, vector, alias)).
		NotTo(gomega.HaveOccurred(), "failed to add alias %s for vector %s", alias, vector)
}

package controller

import (
	"context"
	"fmt"
	"time"

	global "github.com/konfidence-project/crds/api/global/v1alpha1"
	konfcompref "github.com/konfidence-project/pkg/ocm/compref"
	pkgOcm "github.com/konfidence-project/pkg/ocm/repository"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ocmDescriptor "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	"ocm.software/open-component-model/bindings/go/oci/compref"
)

// create reference creates a reference and fails the test in case of errors.
func createReference(component string) compref.Ref {
	ref, err := konfcompref.Parse(fmt.Sprintf("http://%s//%s", registryEndpoint, component))
	ExpectWithOffset(1, err).
		NotTo(HaveOccurred(), "failed to create reference for component %s", component)
	return *ref
}

// pushComponent pushes an artifact descriptor into the OCI registry with an optional alias.
func pushComponent(ctx context.Context, client pkgOcm.Client, ref compref.Ref, alias *string) {
	descriptor := ocmDescriptor.Descriptor{
		Meta: ocmDescriptor.Meta{Version: "v2"},
		Component: ocmDescriptor.Component{
			ComponentMeta: ocmDescriptor.ComponentMeta{
				ObjectMeta: ocmDescriptor.ObjectMeta{
					Name:    ref.Component,
					Version: ref.Version,
				},
			},
			Provider: ocmDescriptor.Provider{Name: "test"},
		},
	}
	ExpectWithOffset(1,
		client.Save(ctx, ref.Repository, descriptor)).
		NotTo(HaveOccurred(), "failed to create component %s", ref)
	if alias != nil {
		ExpectWithOffset(1,
			client.AddAlias(ctx, ref, *alias)).
			NotTo(HaveOccurred(), "failed to add alias %s for component %s", *alias, ref)
	}
}

// pushVector pushes a vector (artifact descriptor with references) into the OCI registry with the given alias.
func pushVector(ctx context.Context, client pkgOcm.Client, vector compref.Ref, artifacts []compref.Ref, alias string) {
	references := make([]ocmDescriptor.Reference, 0, len(artifacts))
	for i, artifact := range artifacts {
		references = append(references, ocmDescriptor.Reference{
			ElementMeta: ocmDescriptor.ElementMeta{
				ObjectMeta: ocmDescriptor.ObjectMeta{
					Name:    fmt.Sprintf("ref-%d", i),
					Version: artifact.Version,
				},
			},
			Component: artifact.Component,
		})
	}
	descriptor := ocmDescriptor.Descriptor{
		Meta: ocmDescriptor.Meta{Version: "v2"},
		Component: ocmDescriptor.Component{
			ComponentMeta: ocmDescriptor.ComponentMeta{
				ObjectMeta: ocmDescriptor.ObjectMeta{
					Name:    vector.Component,
					Version: vector.Version,
				},
			},
			Provider:   ocmDescriptor.Provider{Name: "konfidence"},
			References: references,
		},
	}
	ExpectWithOffset(1,
		client.Save(ctx, vector.Repository, descriptor)).
		NotTo(HaveOccurred(), "failed to create vector %s", vector)
	ExpectWithOffset(1,
		client.AddAlias(ctx, vector, alias)).
		NotTo(HaveOccurred(), "failed to add alias %s for vector %s", alias, vector)
}

// newVectorTemplateCR creates a VectorTemplate CR.
//
//nolint:unparam // namespace is the same in every call, keep as param for consistency
func createVectorTemplateCR(
	ctx context.Context,
	name, namespace string,
	artifacts []compref.Ref,
	vector compref.Ref,
	base *compref.Ref) *global.VectorTemplate {
	components := make([]global.Component, 0, len(artifacts))
	for _, artifact := range artifacts {
		components = append(components, global.Component{
			Name: artifact.String(),
		})
	}
	var baseRef *string
	if base != nil {
		baseRef = new(base.String())
	}
	vectorTemplate := &global.VectorTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: global.VectorTemplateSpec{
			ReconcileInterval: &metav1.Duration{Duration: time.Hour}, // long interval to avoid re-reconciliation during test
			UploadTarget:      vector.String(),
			Base:              baseRef,
			Components:        components,
		},
	}
	Expect(k8sClient.Create(ctx, vectorTemplate)).To(Succeed())
	return vectorTemplate
}

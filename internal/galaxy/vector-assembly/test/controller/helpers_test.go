/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package controller

import (
	"context"
	"fmt"
	"time"

	global "github.com/konfidence-project/crds/api/global/v1alpha1"
	pkgOcm "github.com/konfidence-project/pkg/ocm/repository"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ocmDescriptor "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	"ocm.software/open-component-model/bindings/go/oci/compref"
)

// pushComponent pushes a component descriptor into the Zot registry.
func pushComponent(ctx context.Context, client pkgOcm.Client, endpoint, componentName, version string) {
	ref := mustParseRef(fmt.Sprintf("http://%s//%s", endpoint, componentName))

	descriptor := ocmDescriptor.Descriptor{
		Meta: ocmDescriptor.Meta{Version: "v2"},
		Component: ocmDescriptor.Component{
			ComponentMeta: ocmDescriptor.ComponentMeta{
				ObjectMeta: ocmDescriptor.ObjectMeta{
					Name:    componentName,
					Version: version,
				},
			},
			Provider: ocmDescriptor.Provider{Name: "test"},
		},
	}

	err := client.Save(ctx, ref.Repository, descriptor)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "failed to create mock component %s@%s", componentName, version)
}

// pushVector pushes a vector (component descriptor with references) into the Zot registry.
func pushVector(ctx context.Context, client pkgOcm.Client, endpoint, vectorName, version string, artifacts []vectorArtifact) {
	ref := mustParseRef(fmt.Sprintf("http://%s//%s", endpoint, vectorName))

	references := make([]ocmDescriptor.Reference, 0, len(artifacts))
	for _, artifact := range artifacts {
		references = append(references, ocmDescriptor.Reference{
			ElementMeta: ocmDescriptor.ElementMeta{
				ObjectMeta: ocmDescriptor.ObjectMeta{
					Name:    artifact.Name,
					Version: artifact.Version,
				},
			},
			Component: artifact.Name,
		})
	}

	descriptor := ocmDescriptor.Descriptor{
		Meta: ocmDescriptor.Meta{Version: "v2"},
		Component: ocmDescriptor.Component{
			ComponentMeta: ocmDescriptor.ComponentMeta{
				ObjectMeta: ocmDescriptor.ObjectMeta{
					Name:    vectorName,
					Version: version,
				},
			},
			Provider:   ocmDescriptor.Provider{Name: "konfidence"},
			References: references,
		},
	}

	err := client.Save(ctx, ref.Repository, descriptor)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "failed to create mock vector %s@%s", vectorName, version)
}

// vectorArtifact is a simple struct for mock vector references.
type vectorArtifact struct {
	Name    string
	Version string
}

// getDescriptorFromRegistry reads back a component descriptor from Zot.
func getDescriptorFromRegistry(ctx context.Context, client pkgOcm.Client, endpoint, componentName string) (ocmDescriptor.Descriptor, error) {
	ref := mustParseRef(fmt.Sprintf("http://%s//%s", endpoint, componentName))
	return client.Get(ctx, *ref)
}

// mustParseRef parses a component reference string and panics on error.
func mustParseRef(reference string) *compref.Ref {
	ref, err := compref.Parse(reference)
	ExpectWithOffset(1, err).NotTo(HaveOccurred(), "failed to parse ref: %s", reference)
	return ref
}

// newVectorTemplateCR creates a VectorTemplate CR with the dynamic registry endpoint.
//
//nolint:unparam // namespace is the same in every call, keep as param for consistency
func newVectorTemplateCR(name, namespace, endpoint, vectorName string, componentNames []string, base *string) *global.VectorTemplate {
	components := make([]global.Component, 0, len(componentNames))
	for _, componentName := range componentNames {
		components = append(components, global.Component{
			Name: fmt.Sprintf("http://%s//%s", endpoint, componentName),
		})
	}

	var baseRef *string
	if base != nil {
		baseRefStr := fmt.Sprintf("http://%s//%s", endpoint, *base)
		baseRef = &baseRefStr
	}

	return &global.VectorTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: global.VectorTemplateSpec{
			ReconcileInterval: &metav1.Duration{Duration: time.Hour}, // long interval to avoid re-reconciliation during test
			UploadTarget:      fmt.Sprintf("http://%s//%s", endpoint, vectorName),
			Base:              baseRef,
			Components:        components,
		},
	}
}

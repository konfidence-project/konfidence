package controller_test

import (
	"context"
	"time"

	galaxy "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	"github.com/konfidence-project/konfidence/pkg/testutil/ocm"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"ocm.software/open-component-model/bindings/go/oci/compref"
)

// createReference creates a reference and fails the test in case of errors.
func createReference(component string) compref.Ref {
	return ocm.ParseRef(registryEndpoint, component)
}

// createVectorTemplateCR creates a VectorTemplate CR.
//
//nolint:unparam // namespace is the same in every call, keep as param for consistency
func createVectorTemplateCR(
	ctx context.Context,
	name, namespace string,
	artifacts []compref.Ref,
	vector compref.Ref,
	base *compref.Ref) *galaxy.VectorTemplate {
	components := make([]galaxy.Component, 0, len(artifacts))
	for _, artifact := range artifacts {
		components = append(components, galaxy.Component{
			Name: artifact.String(),
		})
	}
	var baseRef *string
	if base != nil {
		baseRef = new(base.String())
	}
	vectorTemplate := &galaxy.VectorTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: galaxy.VectorTemplateSpec{
			ReconcileInterval: &metav1.Duration{Duration: time.Hour}, // long interval to avoid re-reconciliation during test
			UploadTarget:      vector.String(),
			Base:              baseRef,
			Components:        components,
		},
	}
	Expect(k8sClient.Create(ctx, vectorTemplate)).To(Succeed())
	return vectorTemplate
}

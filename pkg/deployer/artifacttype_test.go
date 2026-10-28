package deployer

import (
	konfidencev1alpha1 "github.com/konfidence-project/konfidence/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

var _ = Describe("ArtifactTypePredicate", func() {
	It("accepts an ArtifactDeployment with the matching artifact type", func() {
		deployment := &konfidencev1alpha1.ArtifactDeployment{
			Spec: konfidencev1alpha1.ArtifactDeploymentSpec{ArtifactType: konfidencev1alpha1.MustArtifactType("helm.example.com")},
		}

		Expect(ArtifactTypePredicate("helm.example.com").Create(event.CreateEvent{Object: deployment})).To(BeTrue())
	})

	It("rejects an ArtifactDeployment with a different artifact type", func() {
		deployment := &konfidencev1alpha1.ArtifactDeployment{
			Spec: konfidencev1alpha1.ArtifactDeploymentSpec{ArtifactType: konfidencev1alpha1.MustArtifactType("kustomize.example.com")},
		}

		Expect(ArtifactTypePredicate("helm.example.com").Create(event.CreateEvent{Object: deployment})).To(BeFalse())
	})
})

package deployer

import (
	"testing"

	konfidencev1alpha1 "github.com/konfidence-project/konfidence/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func TestArtifactTypePredicate(t *testing.T) {
	helmDeployment := &konfidencev1alpha1.ArtifactDeployment{
		Spec: konfidencev1alpha1.ArtifactDeploymentSpec{Manifest: konfidencev1alpha1.ArtifactManifest{Type: "helm.example.com"}},
	}
	otherDeployment := &konfidencev1alpha1.ArtifactDeployment{
		Spec: konfidencev1alpha1.ArtifactDeploymentSpec{Manifest: konfidencev1alpha1.ArtifactManifest{Type: "kustomize.example.com"}},
	}
	if !ArtifactTypePredicate("helm.example.com").Create(event.CreateEvent{Object: helmDeployment}) {
		t.Fatal("predicate rejected a matching artifact deployment")
	}
	if ArtifactTypePredicate("helm.example.com").Create(event.CreateEvent{Object: otherDeployment}) {
		t.Fatal("predicate accepted a non-matching artifact deployment")
	}
}

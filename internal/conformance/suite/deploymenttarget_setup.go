package suite

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func newDeploymentTarget(environment *Environment, namespace string) *konfidence.DeploymentTarget {
	ginkgo.GinkgoHelper()
	gomega.Expect(environment.Inputs.DeploymentTarget.ArtifactType.Valid()).
		To(gomega.BeTrue(), "baseline artifactType must be valid")
	target := &konfidence.DeploymentTarget{ObjectMeta: metav1.ObjectMeta{Name: "target", Namespace: namespace}}
	environment.Inputs.DeploymentTarget.DeepCopyInto(&target.Spec)
	return target
}

func useMissingParameterResource(target *konfidence.DeploymentTarget) {
	ginkgo.GinkgoHelper()
	if target.Spec.Parameters == nil {
		ginkgo.Skip("baseline has no parameter reference")
	}
	// Derive an invalid reference without guessing provider-specific payloads.
	target.Spec.Parameters.Name = "missing-" + uuid.NewString()
}

func createForeignTargets(
	ctx context.Context, c client.Client, target *konfidence.DeploymentTarget,
) []*konfidence.DeploymentTarget {
	ginkgo.GinkgoHelper()
	_, domain, _ := strings.Cut(string(target.Spec.ArtifactType), ".")
	foreignTypes := []konfidence.ArtifactType{
		konfidence.ArtifactType("conformance-" + uuid.NewString() + "." + domain),
		konfidence.ArtifactType("app." + uuid.NewString() + ".invalid"),
	}
	baselines := make([]*konfidence.DeploymentTarget, 0, len(foreignTypes))
	for i, artifactType := range foreignTypes {
		foreign := &konfidence.DeploymentTarget{ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("foreign-%d", i), Namespace: target.Namespace,
		}}
		target.Spec.DeepCopyInto(&foreign.Spec)
		foreign.Spec.ArtifactType = artifactType
		createDeploymentTarget(ctx, c, foreign)
		baselines = append(baselines, foreign.DeepCopy())
	}
	return baselines
}

func createDeploymentTarget(ctx context.Context, c client.Client, target *konfidence.DeploymentTarget) {
	ginkgo.GinkgoHelper()
	if err := c.Create(ctx, target); err != nil {
		ginkgo.Fail("cannot create DeploymentTarget")
	}
}

func readDeploymentTarget(
	ctx context.Context, g gomega.Gomega, c client.Client, target *konfidence.DeploymentTarget,
) *konfidence.DeploymentTarget {
	ginkgo.GinkgoHelper()
	current := &konfidence.DeploymentTarget{}
	err := c.Get(ctx, client.ObjectKeyFromObject(target), current)
	if err != nil {
		// Replace API errors rather than letting Gomega print response bodies.
		err = fmt.Errorf("cannot read DeploymentTarget")
	}
	g.Expect(err).To(gomega.Succeed())
	return current
}

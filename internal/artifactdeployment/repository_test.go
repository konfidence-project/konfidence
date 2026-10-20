package artifactdeployment_test

import (
	"context"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/artifactdeployment"
	landscapedomain "github.com/konfidence-project/konfidence/internal/landscape"
	pkgctrl "github.com/konfidence-project/konfidence/pkg/controller"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	landscapeID        = "dev"
	vectorDeploymentID = "vd-a"
	stageID            = "stage-dev"
	landscapeNamespace = "kden-l-dev"
)

func projectScheme() *runtime.Scheme {
	GinkgoHelper()
	scheme := runtime.NewScheme()
	Expect(konfidence.AddToScheme(scheme)).To(Succeed())
	return scheme
}

func repositoryWith(objects ...client.Object) artifactdeployment.Repository {
	GinkgoHelper()
	k8s := fake.NewClientBuilder().WithScheme(projectScheme()).WithObjects(objects...).Build()
	return artifactdeployment.NewRepository(k8s)
}

func stageVersionResource(name, landscapeNamespace, stageName string) *konfidence.StageVersion {
	return &konfidence.StageVersion{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: landscapeNamespace},
		Spec: konfidence.StageVersionSpec{
			StageRef: &konfidence.StageReference{Name: stageName},
		},
	}
}

func vectorDeploymentResource(name, landscapeNamespace, stageVersionName string) *konfidence.VectorDeployment {
	return &konfidence.VectorDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: landscapeNamespace,
			OwnerReferences: []metav1.OwnerReference{{
				Kind: konfidence.StageVersionKind,
				Name: stageVersionName,
			}},
		},
	}
}

func artifactDeploymentResource(name, landscapeNamespace, landscapeId, vectorDeploymentID string) *konfidence.ArtifactDeployment {
	return &konfidence.ArtifactDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: landscapeNamespace,
			Labels: map[string]string{
				pkgctrl.LandscapeNameLabel: landscapeId,
			},
			OwnerReferences: []metav1.OwnerReference{{
				Kind: konfidence.VectorDeploymentKind,
				Name: vectorDeploymentID,
			}},
		},
	}
}

func scopedLandscapeResource(name, namespace string) landscapedomain.ScopedLandscape {
	return landscapedomain.ScopedLandscape{
		Landscape: konfidence.Landscape{
			ObjectMeta: metav1.ObjectMeta{Name: name},
		},
		Namespace: namespace,
	}
}

func scopedLandscape() landscapedomain.ScopedLandscape {
	return scopedLandscapeResource(landscapeID, landscapeNamespace)
}

var _ = Describe("Repository", func() {

	Describe("ListForScope", func() {
		It("lists without filters", func() {
			resolved, err := repositoryWith(
				stageVersionResource("sv-1", landscapeNamespace, stageID),
				vectorDeploymentResource(vectorDeploymentID, landscapeNamespace, "sv-1"),
				artifactDeploymentResource("ad-a", landscapeNamespace, landscapeID, vectorDeploymentID),
			).ListForScope(context.Background(), []landscapedomain.ScopedLandscape{scopedLandscape()})
			Expect(err).NotTo(HaveOccurred())
			Expect(resolved).To(ConsistOf(And(
				HaveField("ArtifactDeployment.Name", "ad-a"),
				HaveField("StageIds", []string{stageID}),
			)))
		})

		It("filters by landscapes", func() {
			const prodNamespace = "kden-l-prod"
			resolved, err := repositoryWith(
				stageVersionResource("sv-dev", landscapeNamespace, stageID),
				stageVersionResource("sv-prod", prodNamespace, "stage-prod"),
				stageVersionResource("sv-hidden", "kden-l-hidden", "stage-hidden"),
				vectorDeploymentResource("vd-dev", landscapeNamespace, "sv-dev"),
				vectorDeploymentResource("vd-prod", prodNamespace, "sv-prod"),
				vectorDeploymentResource("vd-hidden", "kden-l-hidden", "sv-hidden"),
				artifactDeploymentResource("ad-dev", landscapeNamespace, "dev", "vd-dev"),
				artifactDeploymentResource("ad-prod", prodNamespace, "prod", "vd-prod"),
				artifactDeploymentResource("ad-hidden", "kden-l-hidden", "hidden", "vd-hidden"),
			).ListForScope(context.Background(), []landscapedomain.ScopedLandscape{
				scopedLandscapeResource("dev", landscapeNamespace),
				scopedLandscapeResource("prod", prodNamespace),
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(resolved).To(ConsistOf(
				And(
					HaveField("ArtifactDeployment.Name", "ad-dev"),
					HaveField("LandscapeId", "dev"),
				),
				And(
					HaveField("ArtifactDeployment.Name", "ad-prod"),
					HaveField("LandscapeId", "prod"),
				),
			))
		})

		It("only lists deployments from the supplied landscape scope", func() {
			const prodNamespace = "kden-l-prod"

			resolved, err := repositoryWith(
				stageVersionResource("sv-dev", landscapeNamespace, stageID),
				stageVersionResource("sv-prod", prodNamespace, "stage-prod"),
				vectorDeploymentResource("vd-dev", landscapeNamespace, "sv-dev"),
				vectorDeploymentResource("vd-prod", prodNamespace, "sv-prod"),
				artifactDeploymentResource("ad-dev", landscapeNamespace, "dev", "vd-dev"),
				artifactDeploymentResource("ad-prod", prodNamespace, "prod", "vd-prod"),
			).ListForScope(context.Background(), []landscapedomain.ScopedLandscape{
				scopedLandscapeResource("dev", landscapeNamespace),
			})

			Expect(err).NotTo(HaveOccurred())
			Expect(resolved).To(ConsistOf(And(
				HaveField("ArtifactDeployment.Name", "ad-dev"),
				HaveField("LandscapeId", "dev"),
			)))
		})

		It("skips a provisioning landscape without a namespace", func() {
			resolved, err := repositoryWith().ListForScope(
				context.Background(),
				[]landscapedomain.ScopedLandscape{
					scopedLandscapeResource("provisioning", ""),
				},
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(resolved).To(BeEmpty())
		})

		It("filters by vector deployment", func() {
			shared := artifactDeploymentResource("ad-shared", landscapeNamespace, landscapeID, "vd-a")
			shared.OwnerReferences = append(shared.OwnerReferences, metav1.OwnerReference{
				Kind: konfidence.VectorDeploymentKind,
				Name: "vd-b",
			})
			resolved, err := repositoryWith(
				stageVersionResource("sv-1", landscapeNamespace, stageID),
				vectorDeploymentResource("vd-a", landscapeNamespace, "sv-1"),
				vectorDeploymentResource("vd-b", landscapeNamespace, "sv-1"),
				shared,
				artifactDeploymentResource("ad-only-a", landscapeNamespace, landscapeID, "vd-a"),
			).ListForScope(
				context.Background(),
				[]landscapedomain.ScopedLandscape{scopedLandscape()},
				artifactdeployment.WithVectorDeploymentId("vd-b"),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(resolved).To(ConsistOf(And(
				HaveField("ArtifactDeployment.Name", "ad-shared"),
				HaveField("VectorDeploymentIds", []string{"vd-a", "vd-b"}),
			)))
		})

		It("filters by landscape and vector deployment", func() {
			resolved, err := repositoryWith(
				stageVersionResource("sv-1", landscapeNamespace, stageID),
				vectorDeploymentResource(vectorDeploymentID, landscapeNamespace, "sv-1"),
				vectorDeploymentResource("vd-b", landscapeNamespace, "sv-1"),
				artifactDeploymentResource("ad-match", landscapeNamespace, landscapeID, vectorDeploymentID),
				artifactDeploymentResource("ad-wrong-vd", landscapeNamespace, landscapeID, "vd-b"),
			).ListForScope(
				context.Background(),
				[]landscapedomain.ScopedLandscape{scopedLandscape()},
				artifactdeployment.WithVectorDeploymentId(vectorDeploymentID),
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(resolved).To(ConsistOf(HaveField("ArtifactDeployment.Name", "ad-match")))
		})

		It("resolves stage IDs", func() {
			deployment := artifactDeploymentResource("ad-a", landscapeNamespace, landscapeID, vectorDeploymentID)
			deployment.OwnerReferences = append(deployment.OwnerReferences, metav1.OwnerReference{
				Kind: konfidence.VectorDeploymentKind,
				Name: "vd-b",
			})
			resolved, err := repositoryWith(
				stageVersionResource("sv-1", landscapeNamespace, stageID),
				stageVersionResource("sv-2", landscapeNamespace, "stage-preview"),
				vectorDeploymentResource(vectorDeploymentID, landscapeNamespace, "sv-1"),
				vectorDeploymentResource("vd-b", landscapeNamespace, "sv-2"),
				deployment,
			).ListForScope(
				context.Background(),
				[]landscapedomain.ScopedLandscape{scopedLandscape()},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(resolved).To(ConsistOf(HaveField("StageIds", []string{stageID, "stage-preview"})))
		})

		It("resolves stage and vector deployment IDs", func() {
			resolved, err := repositoryWith(
				stageVersionResource("sv-1", landscapeNamespace, stageID),
				vectorDeploymentResource(vectorDeploymentID, landscapeNamespace, "sv-1"),
				artifactDeploymentResource("ad-a", landscapeNamespace, landscapeID, vectorDeploymentID),
			).ListForScope(
				context.Background(),
				[]landscapedomain.ScopedLandscape{scopedLandscape()},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(resolved).To(ConsistOf(And(
				HaveField("StageIds", []string{stageID}),
				HaveField("VectorDeploymentIds", []string{vectorDeploymentID}),
			)))
		})

		It("resolves vector deployment IDs from owner references", func() {
			deployment := artifactDeploymentResource("ad-a", landscapeNamespace, landscapeID, vectorDeploymentID)
			deployment.OwnerReferences = append(deployment.OwnerReferences, metav1.OwnerReference{
				Kind: konfidence.VectorDeploymentKind,
				Name: "vd-b",
			})
			resolved, err := repositoryWith(
				stageVersionResource("sv-1", landscapeNamespace, stageID),
				stageVersionResource("sv-2", landscapeNamespace, "stage-prod"),
				vectorDeploymentResource(vectorDeploymentID, landscapeNamespace, "sv-1"),
				vectorDeploymentResource("vd-b", landscapeNamespace, "sv-2"),
				deployment,
			).ListForScope(
				context.Background(),
				[]landscapedomain.ScopedLandscape{scopedLandscape()},
			)
			Expect(err).NotTo(HaveOccurred())
			Expect(resolved).To(ConsistOf(HaveField("VectorDeploymentIds", []string{vectorDeploymentID, "vd-b"})))
		})
	})
	Describe("GetForScope", func() {
		It("returns the deployment with its landscape and resolved IDs", func() {
			scoped := scopedLandscape()
			got, err := repositoryWith(
				stageVersionResource("sv-1", landscapeNamespace, stageID),
				vectorDeploymentResource(vectorDeploymentID, landscapeNamespace, "sv-1"),
				artifactDeploymentResource("ad-a", landscapeNamespace, landscapeID, vectorDeploymentID),
			).GetForScope(context.Background(), scoped, "ad-a")
			Expect(err).NotTo(HaveOccurred())
			Expect(got.ArtifactDeployment.Name).To(Equal("ad-a"))
			Expect(got.LandscapeId).To(Equal(landscapeID))
			Expect(got.VectorDeploymentIds).To(Equal([]string{vectorDeploymentID}))
			Expect(got.StageIds).To(Equal([]string{stageID}))
		})

		It("returns ErrNotFound for a missing deployment", func() {
			scoped := scopedLandscape()
			_, err := repositoryWith().GetForScope(context.Background(), scoped, "missing")
			Expect(err).To(MatchError(artifactdeployment.ErrArtifactDeploymentNotFound))
		})

		It("does not return a deployment from another landscape namespace", func() {
			scoped := scopedLandscape()
			_, err := repositoryWith(
				stageVersionResource("sv-1", "kden-l-prod", stageID),
				vectorDeploymentResource(vectorDeploymentID, "kden-l-prod", "sv-1"),
				artifactDeploymentResource("ad-a", "kden-l-prod", landscapeID, vectorDeploymentID),
			).GetForScope(context.Background(), scoped, "ad-a")
			Expect(err).To(MatchError(artifactdeployment.ErrArtifactDeploymentNotFound))
		})

		It("resolves multiple vector deployment and stage IDs", func() {
			scoped := scopedLandscape()
			deployment := artifactDeploymentResource("ad-a", landscapeNamespace, landscapeID, "vd-a")
			deployment.OwnerReferences = append(deployment.OwnerReferences, metav1.OwnerReference{
				Kind: konfidence.VectorDeploymentKind,
				Name: "vd-b",
			})
			got, err := repositoryWith(
				stageVersionResource("sv-1", landscapeNamespace, "stage-a"),
				stageVersionResource("sv-2", landscapeNamespace, "stage-b"),
				vectorDeploymentResource("vd-a", landscapeNamespace, "sv-1"),
				vectorDeploymentResource("vd-b", landscapeNamespace, "sv-2"),
				deployment,
			).GetForScope(context.Background(), scoped, "ad-a")
			Expect(err).NotTo(HaveOccurred())
			Expect(got.VectorDeploymentIds).To(ConsistOf("vd-a", "vd-b"))
			Expect(got.StageIds).To(ConsistOf("stage-a", "stage-b"))
		})
	})
})

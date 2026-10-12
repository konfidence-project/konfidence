package artifactdeployment_test

import (
	"context"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/artifactdeployment"
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
	projectNamespace   = "kden-project"
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

func landscapeResource(name, namespace string) *konfidence.Landscape {
	return &konfidence.Landscape{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: projectNamespace},
		Status:     konfidence.LandscapeStatus{Namespace: namespace},
	}
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

var _ = Describe("Repository", func() {
	It("gets an artifact deployment", func() {
		want := &konfidence.ArtifactDeployment{ObjectMeta: metav1.ObjectMeta{Name: "my-project"}}

		got, err := repositoryWith(want).Get(context.Background(), want.Name)
		Expect(err).NotTo(HaveOccurred())
		Expect(got.Name).To(Equal(want.Name))
	})

	It("returns not found when getting a missing artifact deployment", func() {
		_, err := repositoryWith().Get(context.Background(), "missing")
		Expect(err).To(MatchError(artifactdeployment.ErrNotFound))
	})

	Describe("ListForScope", func() {
		It("lists without filters", func() {
			resolved, err := repositoryWith(
				landscapeResource(landscapeID, landscapeNamespace),
				stageVersionResource("sv-1", landscapeNamespace, stageID),
				vectorDeploymentResource(vectorDeploymentID, landscapeNamespace, "sv-1"),
				artifactDeploymentResource("ad-a", landscapeNamespace, landscapeID, vectorDeploymentID),
			).ListForScope(context.Background(), projectNamespace)
			Expect(err).NotTo(HaveOccurred())
			Expect(resolved).To(ConsistOf(And(
				HaveField("ArtifactDeployment.Name", "ad-a"),
				HaveField("StageIds", []string{stageID}),
			)))
		})

		It("filters by landscape", func() {
			resolved, err := repositoryWith(
				landscapeResource("dev", landscapeNamespace),
				landscapeResource("prod", "kden-l-prod"),
				stageVersionResource("sv-1", landscapeNamespace, stageID),
				stageVersionResource("sv-2", "kden-l-prod", "stage-prod"),
				vectorDeploymentResource(vectorDeploymentID, landscapeNamespace, "sv-1"),
				vectorDeploymentResource(vectorDeploymentID, "kden-l-prod", "sv-2"),
				artifactDeploymentResource("ad-dev", landscapeNamespace, "dev", vectorDeploymentID),
				artifactDeploymentResource("ad-prod", "kden-l-prod", "prod", vectorDeploymentID),
			).ListForScope(context.Background(), projectNamespace, artifactdeployment.WithLandscapeId("dev"))
			Expect(err).NotTo(HaveOccurred())
			Expect(resolved).To(ConsistOf(And(
				HaveField("ArtifactDeployment.Name", "ad-dev"),
				HaveField("LandscapeId", "dev"),
			)))
		})

		It("filters by vector deployment", func() {
			shared := artifactDeploymentResource("ad-shared", landscapeNamespace, landscapeID, "vd-a")
			shared.OwnerReferences = append(shared.OwnerReferences, metav1.OwnerReference{
				Kind: konfidence.VectorDeploymentKind,
				Name: "vd-b",
			})
			resolved, err := repositoryWith(
				landscapeResource(landscapeID, landscapeNamespace),
				stageVersionResource("sv-1", landscapeNamespace, stageID),
				vectorDeploymentResource("vd-a", landscapeNamespace, "sv-1"),
				vectorDeploymentResource("vd-b", landscapeNamespace, "sv-1"),
				shared,
				artifactDeploymentResource("ad-only-a", landscapeNamespace, landscapeID, "vd-a"),
			).ListForScope(context.Background(), projectNamespace, artifactdeployment.WithVectorDeploymentId("vd-b"))
			Expect(err).NotTo(HaveOccurred())
			Expect(resolved).To(ConsistOf(And(
				HaveField("ArtifactDeployment.Name", "ad-shared"),
				HaveField("VectorDeploymentIds", []string{"vd-a", "vd-b"}),
			)))
		})

		It("filters by landscape and vector deployment", func() {
			resolved, err := repositoryWith(
				landscapeResource(landscapeID, landscapeNamespace),
				stageVersionResource("sv-1", landscapeNamespace, stageID),
				vectorDeploymentResource(vectorDeploymentID, landscapeNamespace, "sv-1"),
				vectorDeploymentResource("vd-b", landscapeNamespace, "sv-1"),
				artifactDeploymentResource("ad-match", landscapeNamespace, landscapeID, vectorDeploymentID),
				artifactDeploymentResource("ad-wrong-vd", landscapeNamespace, landscapeID, "vd-b"),
			).ListForScope(
				context.Background(),
				projectNamespace,
				artifactdeployment.WithLandscapeId(landscapeID),
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
				landscapeResource(landscapeID, landscapeNamespace),
				stageVersionResource("sv-1", landscapeNamespace, stageID),
				stageVersionResource("sv-2", landscapeNamespace, "stage-preview"),
				vectorDeploymentResource(vectorDeploymentID, landscapeNamespace, "sv-1"),
				vectorDeploymentResource("vd-b", landscapeNamespace, "sv-2"),
				deployment,
			).ListForScope(context.Background(), projectNamespace)
			Expect(err).NotTo(HaveOccurred())
			Expect(resolved).To(ConsistOf(HaveField("StageIds", []string{stageID, "stage-preview"})))
		})

		It("resolves stage and vector deployment IDs", func() {
			resolved, err := repositoryWith(
				landscapeResource(landscapeID, landscapeNamespace),
				stageVersionResource("sv-1", landscapeNamespace, stageID),
				vectorDeploymentResource(vectorDeploymentID, landscapeNamespace, "sv-1"),
				artifactDeploymentResource("ad-a", landscapeNamespace, landscapeID, vectorDeploymentID),
			).ListForScope(context.Background(), projectNamespace)
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
				landscapeResource(landscapeID, landscapeNamespace),
				stageVersionResource("sv-1", landscapeNamespace, stageID),
				stageVersionResource("sv-2", landscapeNamespace, "stage-prod"),
				vectorDeploymentResource(vectorDeploymentID, landscapeNamespace, "sv-1"),
				vectorDeploymentResource("vd-b", landscapeNamespace, "sv-2"),
				deployment,
			).ListForScope(context.Background(), projectNamespace)
			Expect(err).NotTo(HaveOccurred())
			Expect(resolved).To(ConsistOf(HaveField("VectorDeploymentIds", []string{vectorDeploymentID, "vd-b"})))
		})
	})
})

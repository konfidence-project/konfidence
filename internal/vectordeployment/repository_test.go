package vectordeployment_test

import (
	"context"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	landscapedomain "github.com/konfidence-project/konfidence/internal/landscape"
	"github.com/konfidence-project/konfidence/internal/vectordeployment"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func vectorDeploymentClient(objects ...client.Object) client.Client {
	GinkgoHelper()

	scheme := runtime.NewScheme()
	Expect(konfidence.AddToScheme(scheme)).To(Succeed())

	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objects...).
		Build()
}

//nolint:unparam
func scope(name, projectNamespace, namespace string) landscapedomain.ScopedLandscape {
	return landscapedomain.ScopedLandscape{
		Landscape: konfidence.Landscape{
			ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: projectNamespace},
			Status:     konfidence.LandscapeStatus{Namespace: namespace},
		},
		Namespace: namespace,
	}
}

func vectorDeployment(name, namespace, stageVersionName string) *konfidence.VectorDeployment {
	return &konfidence.VectorDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{{
				Kind: konfidence.StageVersionKind,
				Name: stageVersionName,
			}},
		},
		Spec: konfidence.VectorDeploymentSpec{
			Vector: "https://registry.example.com/ocm//acme.example/vector:1.0.0",
		},
	}
}

func stageVersion(name, namespace, stageId string) *konfidence.StageVersion {
	return &konfidence.StageVersion{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Spec: konfidence.StageVersionSpec{
			Vector:          "https://registry.example.com/ocm//acme.example/vector:1.0.0",
			StageGeneration: 1,
			StageRef:        &konfidence.StageReference{Name: stageId},
		},
	}
}

var _ = Describe("Repository", func() {
	Describe("ListForScope", func() {
		It("lists deployments across the requested landscapes", func() {
			repository := vectordeployment.NewRepository(vectorDeploymentClient(
				vectorDeployment("checkout-v1", "landscape-dev", "checkout-v1"),
				stageVersion("checkout-v1", "landscape-dev", "checkout"),
				vectorDeployment("checkout-v2", "landscape-prod", "checkout-v2"),
				stageVersion("checkout-v2", "landscape-prod", "checkout"),
				vectorDeployment("hidden", "landscape-other", "hidden"),
				stageVersion("hidden", "landscape-other", "hidden"),
			))
			dev := scope("dev", "project-a", "landscape-dev")
			prod := scope("prod", "project-a", "landscape-prod")

			items, err := repository.ListForScope(context.Background(), []landscapedomain.ScopedLandscape{dev, prod})

			Expect(err).NotTo(HaveOccurred())
			Expect(items).To(ConsistOf(
				And(
					HaveField("VectorDeployment.Name", "checkout-v1"),
					HaveField("LandscapeId", "dev"),
					HaveField("StageId", "checkout"),
				),
				And(
					HaveField("VectorDeployment.Name", "checkout-v2"),
					HaveField("LandscapeId", "prod"),
					HaveField("StageId", "checkout"),
				),
			))
		})

		It("limits deployments to the requested landscape", func() {
			repository := vectordeployment.NewRepository(vectorDeploymentClient(
				vectorDeployment("checkout-v1", "landscape-dev", "checkout-v1"),
				stageVersion("checkout-v1", "landscape-dev", "checkout"),
				vectorDeployment("checkout-v2", "landscape-prod", "checkout-v2"),
				stageVersion("checkout-v2", "landscape-prod", "checkout"),
			))

			items, err := repository.ListForScope(
				context.Background(),
				[]landscapedomain.ScopedLandscape{
					scope("dev", "project-a", "landscape-dev"),
				},
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(items).To(ConsistOf(And(
				HaveField("VectorDeployment.Name", "checkout-v1"),
				HaveField("LandscapeId", "dev"),
				HaveField("StageId", "checkout"),
			)))
		})

		It("returns an error when a stage version is missing", func() {
			repository := vectordeployment.NewRepository(vectorDeploymentClient(
				vectorDeployment("checkout-v1", "landscape-dev", "missing"),
			))

			_, err := repository.ListForScope(
				context.Background(),
				[]landscapedomain.ScopedLandscape{
					scope("dev", "project-a", "landscape-dev"),
				},
			)

			Expect(err).To(MatchError(ContainSubstring(
				`resolving stage for vector deployment "checkout-v1"`,
			)))
		})

		It("sorts by landscape and deployment name", func() {
			repository := vectordeployment.NewRepository(
				vectorDeploymentClient(
					vectorDeployment("web", "landscape-prod", "prod-web"),
					vectorDeployment("api", "landscape-prod", "prod-api"),
					vectorDeployment("web", "landscape-dev", "dev-web"),
					vectorDeployment("api", "landscape-dev", "dev-api"),
					stageVersion("prod-web", "landscape-prod", "web"),
					stageVersion("prod-api", "landscape-prod", "api"),
					stageVersion("dev-web", "landscape-dev", "web"),
					stageVersion("dev-api", "landscape-dev", "api"),
				),
			)

			items, err := repository.ListForScope(
				context.Background(),
				[]landscapedomain.ScopedLandscape{
					scope("prod", "project-a", "landscape-prod"),
					scope("dev", "project-a", "landscape-dev"),
				},
			)

			Expect(err).NotTo(HaveOccurred())
			Expect(items).To(HaveExactElements(
				And(
					HaveField("LandscapeId", "dev"),
					HaveField("VectorDeployment.Name", "api"),
				),
				And(
					HaveField("LandscapeId", "dev"),
					HaveField("VectorDeployment.Name", "web"),
				),
				And(
					HaveField("LandscapeId", "prod"),
					HaveField("VectorDeployment.Name", "api"),
				),
				And(
					HaveField("LandscapeId", "prod"),
					HaveField("VectorDeployment.Name", "web"),
				),
			))
		})
	})
	Describe("GetForScope", func() {
		It("returns the deployment with its landscape and stage IDs", func() {
			scoped := scope("dev", "project-a", "landscape-dev")
			got, err := vectordeployment.NewRepository(vectorDeploymentClient(
				vectorDeployment("checkout-v1", "landscape-dev", "checkout-sv"),
				stageVersion("checkout-sv", "landscape-dev", "checkout"),
			)).GetForScope(context.Background(), scoped, "checkout-v1")
			Expect(err).NotTo(HaveOccurred())
			Expect(got.VectorDeployment.Name).To(Equal("checkout-v1"))
			Expect(got.LandscapeId).To(Equal("dev"))
			Expect(got.StageId).To(Equal("checkout"))
		})

		It("returns ErrNotFound for a missing deployment", func() {
			scoped := scope("dev", "project-a", "landscape-dev")
			_, err := vectordeployment.NewRepository(vectorDeploymentClient()).GetForScope(
				context.Background(), scoped, "missing",
			)
			Expect(err).To(MatchError(vectordeployment.ErrVectorDeploymentNotFound))
		})

		It("returns an error when the stage version is missing", func() {
			scoped := scope("dev", "project-a", "landscape-dev")
			_, err := vectordeployment.NewRepository(vectorDeploymentClient(
				vectorDeployment("checkout-v1", "landscape-dev", "missing-sv"),
			)).GetForScope(context.Background(), scoped, "checkout-v1")
			Expect(err).To(HaveOccurred())
			Expect(err).NotTo(MatchError(vectordeployment.ErrVectorDeploymentNotFound))
		})

		It("returns an error when the deployment has no stage version owner", func() {
			noOwner := &konfidence.VectorDeployment{
				ObjectMeta: metav1.ObjectMeta{Name: "no-owner", Namespace: "landscape-dev"},
				Spec:       konfidence.VectorDeploymentSpec{Vector: "https://registry.example.com/ocm//acme.example/vector:1.0.0"},
			}
			scoped := scope("dev", "project-a", "landscape-dev")
			_, err := vectordeployment.NewRepository(vectorDeploymentClient(noOwner)).GetForScope(
				context.Background(), scoped, "no-owner",
			)
			Expect(err).To(MatchError(ContainSubstring(`resolving stage for vector deployment "no-owner"`)))
		})

		It("does not return a deployment from another landscape namespace", func() {
			devScope := scope("dev", "project-a", "landscape-dev")
			_, err := vectordeployment.NewRepository(vectorDeploymentClient(
				vectorDeployment("checkout-v1", "landscape-prod", "checkout-sv"),
				stageVersion("checkout-sv", "landscape-prod", "checkout"),
			)).GetForScope(context.Background(), devScope, "checkout-v1")
			Expect(err).To(MatchError(vectordeployment.ErrVectorDeploymentNotFound))
		})
	})
})

var _ = Describe("StateFromConditions", func() {
	DescribeTable("maps conditions to deployment states",
		func(conditions []metav1.Condition, expected vectordeployment.State) {
			Expect(vectordeployment.StateFromConditions(conditions)).To(Equal(expected))
		},
		Entry("with no conditions", nil, vectordeployment.StateDeployingVector),
		Entry(
			"while deployment is progressing",
			[]metav1.Condition{{
				Type:   konfidence.VectorDownloadedCondition,
				Status: metav1.ConditionTrue,
			}},
			vectordeployment.StateDeployingVector,
		),
		Entry(
			"when deployment is ready",
			[]metav1.Condition{{
				Type:   konfidence.VectorReadyCondition,
				Status: metav1.ConditionTrue,
			}},
			vectordeployment.StateDeploymentReady,
		),
		Entry(
			"when a condition is unmet",
			[]metav1.Condition{{
				Type:   konfidence.VectorDataCreatedCondition,
				Status: metav1.ConditionFalse,
			}},
			vectordeployment.StateDeployingVector,
		),
		Entry(
			"when ready despite an unmet milestone",
			[]metav1.Condition{
				{
					Type:   konfidence.VectorDataCreatedCondition,
					Status: metav1.ConditionFalse,
				},
				{
					Type:   konfidence.VectorReadyCondition,
					Status: metav1.ConditionTrue,
				},
			},
			vectordeployment.StateDeploymentReady,
		),
		Entry(
			"when deployment is stalled",
			[]metav1.Condition{{
				Type:   konfidence.StalledCondition,
				Status: metav1.ConditionTrue,
				Reason: konfidence.VectorDeploymentStalledReasonArtifactDeploymentStalled,
			}},
			vectordeployment.StateDeploymentFailed,
		),
		Entry(
			"not stalled remains deploying",
			[]metav1.Condition{{
				Type:   konfidence.StalledCondition,
				Status: metav1.ConditionFalse,
			}},
			vectordeployment.StateDeployingVector,
		),
		Entry(
			"when stalled outranks ready",
			[]metav1.Condition{
				{
					Type:   konfidence.StalledCondition,
					Status: metav1.ConditionTrue,
				},
				{
					Type:   konfidence.VectorReadyCondition,
					Status: metav1.ConditionTrue,
				},
			},
			vectordeployment.StateDeploymentFailed,
		),
	)
})

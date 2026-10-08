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
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
}

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
	It("lists vector deployments for the scope", func() {
		k8s := vectorDeploymentClient(
			vectorDeployment("checkout-v1", "landscape-dev", "checkout-v1"),
			stageVersion("checkout-v1", "landscape-dev", "checkout"),
			vectorDeployment("checkout-v2", "landscape-prod", "checkout-v2"),
			stageVersion("checkout-v2", "landscape-prod", "checkout"),
			vectorDeployment("hidden", "landscape-other", "hidden"),
			stageVersion("hidden", "landscape-other", "hidden"),
		)
		repository := vectordeployment.NewRepository(k8s)
		dev := scope("dev", "project-a", "landscape-dev")
		prod := scope("prod", "project-a", "landscape-prod")

		items, err := repository.ListForScope(context.Background(), []landscapedomain.ScopedLandscape{dev, prod})
		Expect(err).NotTo(HaveOccurred())
		Expect(items).To(HaveLen(2))

		items, err = repository.ListForScope(context.Background(), []landscapedomain.ScopedLandscape{dev})
		Expect(err).NotTo(HaveOccurred())
		Expect(items).To(ConsistOf(And(
			HaveField("VectorDeployment.Name", "checkout-v1"),
			HaveField("LandscapeId", "dev"),
			HaveField("StageId", "checkout"),
		)))
	})

	It("rejects a missing stage version", func() {
		k8s := vectorDeploymentClient(vectorDeployment("checkout-v1", "landscape-dev", "missing"))

		_, err := vectordeployment.NewRepository(k8s).ListForScope(context.Background(), []landscapedomain.ScopedLandscape{
			scope("dev", "project-a", "landscape-dev"),
		})
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("StateFromConditions", func() {
	DescribeTable("derives the deployment state",
		func(conditions []metav1.Condition, want vectordeployment.State) {
			Expect(vectordeployment.StateFromConditions(conditions)).To(Equal(want))
		},
		Entry("with no conditions", nil, vectordeployment.StateDeployingVector),
		Entry("while deployment is progressing",
			[]metav1.Condition{{
				Type:   konfidence.VectorDownloadedCondition,
				Status: metav1.ConditionTrue,
			}},
			vectordeployment.StateDeployingVector,
		),
		Entry("when deployment is ready",
			[]metav1.Condition{{
				Type:   konfidence.VectorReadyCondition,
				Status: metav1.ConditionTrue,
			}},
			vectordeployment.StateDeploymentReady,
		),
		Entry("when a condition is unmet",
			[]metav1.Condition{{
				Type:   konfidence.VectorDataCreatedCondition,
				Status: metav1.ConditionFalse,
			}},
			vectordeployment.StateDeployingVector,
		),
		Entry("when ready with an unmet milestone",
			[]metav1.Condition{
				{Type: konfidence.VectorDataCreatedCondition, Status: metav1.ConditionFalse},
				{Type: konfidence.VectorReadyCondition, Status: metav1.ConditionTrue},
			},
			vectordeployment.StateDeploymentReady,
		),
	)
})

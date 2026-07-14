package controller

import (
	"context"
	"fmt"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	testocm "github.com/konfidence-project/konfidence/pkg/testutil/ocm"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	eventsv1 "k8s.io/api/events/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	timeout  = time.Second * 35
	interval = time.Millisecond * 250
)

var _ = Describe("Stage Configuration Controller", Ordered, func() {
	const (
		StageConfiguration = "stage-configuration-dev"
		StageDev           = "stage-dev"
		Namespace          = "default"
		TargetNamespace    = "target"
	)

	BeforeEach(func() {
		cleanupResources(context.Background(), k8sClient, Namespace, TargetNamespace)
	})

	AfterEach(func() {
		cleanupResources(context.Background(), k8sClient, Namespace, TargetNamespace)
	})

	Context("When reconciling a stageConfiguration", func() {
		It("should create a Stage with the specific vector version", func() {
			ctx := context.Background()

			vectorRef := testocm.ParseRef(registryEndpoint, "konfidence.cloud/project/constructed-vector:1.0.0")
			artifactRef := testocm.ParseRef(registryEndpoint, "konfidence.cloud/project/constructed-artifact:1.0.0")
			testocm.PushVector(ctx, ocmClient, vectorRef, []compref.Ref{artifactRef}, "sc-stable", testocm.SampleVectorConfig())

			vectorV100 := fmt.Sprintf("http://%s//konfidence.cloud/project/constructed-vector:1.0.0", registryEndpoint)

			createStageConfiguration(ctx, StageConfiguration, StageDev, vectorV100)

			stageConfiguration := &konfidence.StageConfiguration{}
			stageConfigurationLookupKey := types.NamespacedName{Name: StageConfiguration, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageConfigurationLookupKey, stageConfiguration)).To(Succeed())
				g.Expect(stageConfiguration.Spec.Name).To(Equal(StageDev))
				g.Expect(stageConfiguration.Spec.Vector).To(Equal(vectorV100))
				g.Expect(stageConfiguration.Spec.TargetNamespace).To(Equal(TargetNamespace))
				g.Expect(stageConfiguration.Status.Conditions).To(HaveLen(1))
				g.Expect(stageConfiguration.Status.Conditions[0].Type).To(Equal(konfidence.StageConfigurationReadyCondition))
				g.Expect(stageConfiguration.Status.Conditions[0].Status).To(Equal(metav1.ConditionTrue))
			}, timeout, interval).Should(Succeed())

			stage := &konfidence.Stage{}
			stageLookupKey := types.NamespacedName{Name: StageDev, Namespace: TargetNamespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
				g.Expect(stage.Spec.Vector).To(Equal(vectorV100))
				g.Expect(stage.Labels).To(HaveKeyWithValue("app.kubernetes.io/managed-by", stageConfigurationControllerName))
				g.Expect(stage.Labels).To(HaveKey("konfidence.cloud/stage-configuration"))
			}, timeout, interval).Should(Succeed())

			eventList := &eventsv1.EventList{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.List(ctx, eventList, client.InNamespace(Namespace))).To(Succeed())
				matchingEvents := filterEventsByReason(eventList.Items, "StageConfigurationReconciled")
				g.Expect(matchingEvents).ToNot(BeEmpty())
			}, timeout, interval).Should(Succeed())
		})

		It("should update the Stage when the configuration vector changes", func() {
			ctx := context.Background()

			// Push v1.0.1 with a "latest" alias — controller resolves the alias to the real semver
			vectorRef := testocm.ParseRef(registryEndpoint, "konfidence.cloud/project/constructed-vector:1.0.1")
			artifactRef := testocm.ParseRef(registryEndpoint, "konfidence.cloud/project/constructed-artifact:1.0.1")
			testocm.PushVector(ctx, ocmClient, vectorRef, []compref.Ref{artifactRef}, "sc-edge", testocm.SampleVectorConfig())

			vectorV100 := fmt.Sprintf("http://%s//konfidence.cloud/project/constructed-vector:1.0.0", registryEndpoint)
			vectorV101 := fmt.Sprintf("http://%s//konfidence.cloud/project/constructed-vector:1.0.1", registryEndpoint)
			vectorLatest := fmt.Sprintf("http://%s//konfidence.cloud/project/constructed-vector:sc-edge", registryEndpoint)

			createStageConfiguration(ctx, StageConfiguration, StageDev, vectorV100)

			stage := &konfidence.Stage{}
			stageLookupKey := types.NamespacedName{Name: StageDev, Namespace: TargetNamespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
				g.Expect(stage.Spec.Vector).To(Equal(vectorV100))
			}, timeout, interval).Should(Succeed())

			stageConfiguration := &konfidence.StageConfiguration{}
			stageConfigurationLookupKey := types.NamespacedName{Name: StageConfiguration, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageConfigurationLookupKey, stageConfiguration)).To(Succeed())
				stageConfiguration.Spec.Vector = vectorLatest
				g.Expect(k8sClient.Update(ctx, stageConfiguration)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
				g.Expect(stage.Spec.Vector).To(Equal(vectorV101))
			}, timeout, interval).Should(Succeed())
		})

		It("should delete the Stage and release the finalizer when the StageConfiguration is deleted", func() {
			ctx := context.Background()

			vectorRef := testocm.ParseRef(registryEndpoint, "konfidence.cloud/project/constructed-vector:2.0.0")
			artifactRef := testocm.ParseRef(registryEndpoint, "konfidence.cloud/project/constructed-artifact:2.0.0")
			testocm.PushVector(ctx, ocmClient, vectorRef, []compref.Ref{artifactRef}, "sc-delete", testocm.SampleVectorConfig())

			vectorV200 := fmt.Sprintf("http://%s//konfidence.cloud/project/constructed-vector:2.0.0", registryEndpoint)

			createStageConfiguration(ctx, StageConfiguration, StageDev, vectorV200)

			stage := &konfidence.Stage{}
			stageLookupKey := types.NamespacedName{Name: StageDev, Namespace: TargetNamespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			stageConfiguration := &konfidence.StageConfiguration{}
			stageConfigurationLookupKey := types.NamespacedName{Name: StageConfiguration, Namespace: Namespace}
			Expect(k8sClient.Get(ctx, stageConfigurationLookupKey, stageConfiguration)).To(Succeed())
			Expect(k8sClient.Delete(ctx, stageConfiguration)).To(Succeed())

			// The managed Stage is deleted and the StageConfiguration finalizer released.
			Eventually(func(g Gomega) {
				err := k8sClient.Get(ctx, stageLookupKey, stage)
				g.Expect(apierrors.IsNotFound(err)).To(BeTrue())

				err = k8sClient.Get(ctx, stageConfigurationLookupKey, stageConfiguration)
				g.Expect(apierrors.IsNotFound(err)).To(BeTrue())
			}, timeout, interval).Should(Succeed())
		})

		It("should recreate the Stage if it is deleted while the StageConfiguration exists", func() {
			ctx := context.Background()

			vectorRef := testocm.ParseRef(registryEndpoint, "konfidence.cloud/project/constructed-vector:3.0.0")
			artifactRef := testocm.ParseRef(registryEndpoint, "konfidence.cloud/project/constructed-artifact:3.0.0")
			testocm.PushVector(ctx, ocmClient, vectorRef, []compref.Ref{artifactRef}, "sc-drift", testocm.SampleVectorConfig())

			vectorV300 := fmt.Sprintf("http://%s//konfidence.cloud/project/constructed-vector:3.0.0", registryEndpoint)

			createStageConfiguration(ctx, StageConfiguration, StageDev, vectorV300)

			stage := &konfidence.Stage{}
			stageLookupKey := types.NamespacedName{Name: StageDev, Namespace: TargetNamespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageLookupKey, stage)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			// Delete the Stage out from under the controller; the periodic reconcile
			// (defaultReconcileInterval) recreates it, so recovery can take up to one
			// interval.
			Expect(k8sClient.Delete(ctx, stage)).To(Succeed())

			Eventually(func(g Gomega) {
				recreated := &konfidence.Stage{}
				g.Expect(k8sClient.Get(ctx, stageLookupKey, recreated)).To(Succeed())
				g.Expect(recreated.UID).ToNot(Equal(stage.UID))
				g.Expect(recreated.Spec.Vector).To(Equal(vectorV300))
			}, timeout, interval).Should(Succeed())
		})

		It("should mark not ready when the credential Secret does not exist", func() {
			ctx := context.Background()

			createPKIStageConfiguration(ctx, "sc-missing-secret", "sc-missing-stage",
				fmt.Sprintf("http://%s//konfidence.cloud/project/constructed-vector:missing", registryEndpoint),
				scCredentials([]string{"missing-secret"}), nil)

			assertSCReady(ctx, "sc-missing-secret", metav1.ConditionFalse)

			// The failure reason is surfaced on the Ready condition message.
			sc := &konfidence.StageConfiguration{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "sc-missing-secret", Namespace: Namespace}, sc)).To(Succeed())
			cond := apimeta.FindStatusCondition(sc.Status.Conditions, konfidence.StageConfigurationReadyCondition)
			Expect(cond).NotTo(BeNil())
			Expect(cond.Message).NotTo(BeEmpty())
		})
	})
})

func filterEventsByReason(events []eventsv1.Event, reason string) []eventsv1.Event {
	var filtered []eventsv1.Event
	for _, e := range events {
		if e.Reason == reason {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

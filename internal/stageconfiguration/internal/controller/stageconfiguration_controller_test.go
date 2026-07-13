package controller

import (
	"context"
	"fmt"
	"reflect"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/stageconfiguration/internal/template"
	testocm "github.com/konfidence-project/konfidence/pkg/testutil/ocm"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	eventsv1 "k8s.io/api/events/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Stage Configuration Controller", Ordered, func() {
	const (
		StageConfiguration = "stage-configuration-dev"
		StageDev           = "stage-dev"
		StageSyncDev       = "sync-stage-dev"
		Namespace          = "default"
		TargetNamespace    = "target"
		timeout            = time.Second * 30
		interval           = time.Millisecond * 250
	)

	BeforeEach(func() {
		cleanupResources(context.Background(), k8sClient, Namespace, TargetNamespace)
	})

	AfterEach(func() {
		cleanupResources(context.Background(), k8sClient, Namespace, TargetNamespace)
	})

	Context("When reconciling a stageConfiguration", func() {
		It("should successfully create a stageSync with specific vector version", func() {
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

			stageTemplate := createStageTemplate(*stageConfiguration, vectorV100)
			var originalTemplate template.StageTemplate

			stageSync := &konfidence.StageSync{}
			stageSyncLookupKey := types.NamespacedName{Name: StageSyncDev, Namespace: TargetNamespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageSyncLookupKey, stageSync)).To(Succeed())
				g.Expect(stageSync.Name).To(Equal(StageSyncDev))
				g.Expect(json.Unmarshal(stageSync.Spec.StageTemplate.Raw, &originalTemplate)).To(Succeed())
				g.Expect(reflect.DeepEqual(stageTemplate, originalTemplate)).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			eventList := &eventsv1.EventList{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.List(ctx, eventList, client.InNamespace(Namespace))).To(Succeed())
				matchingEvents := filterEventsByReason(eventList.Items, "StageConfigurationReconciled")
				g.Expect(matchingEvents).ToNot(BeEmpty())
			}, timeout, interval).Should(Succeed())
		})

		It("should successfully update an existing stage with latest vector version", func() {
			ctx := context.Background()

			// Push v1.0.1 with a "latest" alias — controller resolves the alias to the real semver
			vectorRef := testocm.ParseRef(registryEndpoint, "konfidence.cloud/project/constructed-vector:1.0.1")
			artifactRef := testocm.ParseRef(registryEndpoint, "konfidence.cloud/project/constructed-artifact:1.0.1")
			testocm.PushVector(ctx, ocmClient, vectorRef, []compref.Ref{artifactRef}, "sc-edge", testocm.SampleVectorConfig())

			vectorV100 := fmt.Sprintf("http://%s//konfidence.cloud/project/constructed-vector:1.0.0", registryEndpoint)
			vectorV101 := fmt.Sprintf("http://%s//konfidence.cloud/project/constructed-vector:1.0.1", registryEndpoint)
			vectorLatest := fmt.Sprintf("http://%s//konfidence.cloud/project/constructed-vector:sc-edge", registryEndpoint)

			createStageSync(ctx, k8sClient, StageSyncDev, Namespace, StageConfiguration, TargetNamespace, StageDev, vectorV100)

			var stageTemplate template.StageTemplate

			stageSync := &konfidence.StageSync{}
			stageSyncLookupKey := types.NamespacedName{Name: StageSyncDev, Namespace: TargetNamespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageSyncLookupKey, stageSync)).To(Succeed())
				g.Expect(stageSync.Name).To(Equal(StageSyncDev))
				g.Expect(json.Unmarshal(stageSync.Spec.StageTemplate.Raw, &stageTemplate)).To(Succeed())
				g.Expect(stageTemplate.Spec.Vector).To(Equal(vectorV100))
			}, timeout, interval).Should(Succeed())

			createStageConfiguration(ctx, StageConfiguration, StageDev, vectorLatest)

			stageConfiguration := &konfidence.StageConfiguration{}
			stageConfigurationLookupKey := types.NamespacedName{Name: StageConfiguration, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageConfigurationLookupKey, stageConfiguration)).To(Succeed())
				g.Expect(stageConfiguration.Spec.Name).To(Equal(StageDev))
				g.Expect(stageConfiguration.Spec.Vector).To(Equal(vectorLatest))
				g.Expect(stageConfiguration.Spec.TargetNamespace).To(Equal(TargetNamespace))
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageSyncLookupKey, stageSync)).To(Succeed())
				g.Expect(json.Unmarshal(stageSync.Spec.StageTemplate.Raw, &stageTemplate)).To(Succeed())
				g.Expect(stageTemplate.Spec.Vector).To(Equal(vectorV101))
			}, timeout, interval).Should(Succeed())

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageConfigurationLookupKey, stageConfiguration)).To(Succeed())
				g.Expect(stageConfiguration.Status.Conditions).To(HaveLen(1))
				g.Expect(stageConfiguration.Status.Conditions[0].Type).To(Equal(konfidence.StageConfigurationReadyCondition))
				g.Expect(stageConfiguration.Status.Conditions[0].Status).To(Equal(metav1.ConditionTrue))
			}, timeout, interval).Should(Succeed())

			eventList := &eventsv1.EventList{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.List(ctx, eventList, client.InNamespace(Namespace))).To(Succeed())
				matchingEvents := filterEventsByReason(eventList.Items, "StageConfigurationReconciled")
				g.Expect(matchingEvents).ToNot(BeEmpty())
			}, timeout, interval).Should(Succeed())
		})
		It("should mark not ready when the credential Secret does not exist", func() {
			ctx := context.Background()

			createPKIStageConfiguration(ctx, "sc-missing-secret", "sc-missing-stage",
				fmt.Sprintf("http://%s//konfidence.cloud/project/constructed-vector:missing", registryEndpoint),
				scCredentials([]string{"missing-secret"}), nil)

			assertSCReady(ctx, "sc-missing-secret", metav1.ConditionFalse)
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

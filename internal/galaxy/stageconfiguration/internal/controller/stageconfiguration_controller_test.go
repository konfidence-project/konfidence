package controller

import (
	"context"
	"reflect"
	"time"

	global "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/galaxy/stageconfiguration/internal/ports"
	"github.com/konfidence-project/konfidence/internal/galaxy/stageconfiguration/internal/ports/mocks"
	"github.com/konfidence-project/konfidence/internal/galaxy/stageconfiguration/internal/template"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	pkgOcm "github.com/konfidence-project/konfidence/pkg/ocm/repository"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/mock/gomock"
	v1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Stage Configuration Controller", Ordered, func() {
	const (
		StageConfiguration = "stage-configuration-dev"
		StageDev           = "stage-dev"
		StageSyncDev       = "sync-stage-dev"
		V100               = "1.0.0"
		V101               = "1.0.1"
		VectorLatest       = "http://localhost:5100//konfidence.cloud/project/constructed-vector:latest"
		VectorV100         = "http://localhost:5100//konfidence.cloud/project/constructed-vector" + ":" + V100
		VectorV101         = "http://localhost:5100//konfidence.cloud/project/constructed-vector" + ":" + V101
		Namespace          = "default"
		TargetNamespace    = "target"
		timeout            = time.Second * 10
		interval           = time.Millisecond * 250
	)

	var (
		reconciler     *StageConfigurationReconciler
		vectorPortMock *mocks.MockVectorPort
		mockCtrl       *gomock.Controller
	)

	BeforeAll(func() {
		// mock setup
		mockCtrl = gomock.NewController(GinkgoT())
		vectorPortMock = mocks.NewMockVectorPort(mockCtrl)

		reconciler = &StageConfigurationReconciler{
			Mgr: k8sManager,
			OcmClientProvider: pkgOcm.ClientProviderFunc(func(
				ctx context.Context, k8sClient client.Reader, namespace string,
				credentialsConfig []global.CredentialsConfig,
			) (pkgOcm.Client, error) {
				return nil, nil
			}),
			VectorPortProvider: ports.VectorPortProviderFunc(func(verifier crypto.Verifier, client pkgOcm.Client) ports.VectorPort {
				return vectorPortMock
			}),
			Scheme: k8sScheme,
		}
		err := reconciler.SetupWithManager(k8sManager)
		Expect(err).ToNot(HaveOccurred())
	})

	AfterAll(func() {
		if mockCtrl != nil {
			mockCtrl.Finish()
		}
	})

	BeforeEach(func() {
		CleanupResources(context.Background(), k8sClient, Namespace, TargetNamespace)
	})

	AfterEach(func() {
		CleanupResources(context.Background(), k8sClient, Namespace, TargetNamespace)
	})

	Context("When reconciling a stageConfiguration", func() {
		It("should successfully create a stageSync with specific vector version ", func() {
			ctx := context.Background()
			vectorPortMock.EXPECT().GetLatestVectorVersion(gomock.Any(), VectorV100).Return(VectorV100, nil)
			// create target namespace
			// note: since test env cannot delete namespaces the target namespace is created once in the first test
			CreateNamespace(ctx, k8sClient, TargetNamespace)
			ns := &v1.Namespace{}
			nsLookupKey := client.ObjectKey{
				Name: TargetNamespace,
			}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, nsLookupKey, ns)).To(Succeed())
			}, timeout, interval).Should(Succeed())

			// create stageConfiguration
			CreateStageConfiguration(ctx, k8sClient, StageConfiguration, Namespace, TargetNamespace, StageDev, VectorV100)

			// check that the stageConfiguration has been created and has valid properties
			stageConfiguration := &global.StageConfiguration{}
			stageConfigurationLookupKey := types.NamespacedName{Name: StageConfiguration, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageConfigurationLookupKey, stageConfiguration)).To(Succeed())
				g.Expect(stageConfiguration.Spec.Name).To(Equal(StageDev))
				g.Expect(stageConfiguration.Spec.Vector).To(Equal(VectorV100))
				g.Expect(stageConfiguration.Spec.TargetNamespace).To(Equal(TargetNamespace))
				g.Expect(stageConfiguration.Status.Conditions).To(HaveLen(1))
				g.Expect(stageConfiguration.Status.Conditions[0].Type).To(Equal(global.StageConfigurationReadyCondition))
				g.Expect(stageConfiguration.Status.Conditions[0].Status).To(Equal(metav1.ConditionTrue))
			}, timeout, interval).Should(Succeed())

			stageTemplate := CreateStageTemplate(*stageConfiguration, VectorV100)
			var originalTemplate template.StageTemplate

			// check that the stageSync has been created and has valid properties
			stageSync := &global.StageSync{}
			stageSyncLookupKey := types.NamespacedName{Name: StageSyncDev, Namespace: TargetNamespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageSyncLookupKey, stageSync)).To(Succeed())
				g.Expect(stageSync.Name).To(Equal(StageSyncDev))
				g.Expect(json.Unmarshal(stageSync.Spec.StageTemplate.Raw, &originalTemplate)).To(Succeed())
				g.Expect(reflect.DeepEqual(stageTemplate, originalTemplate)).To(BeTrue())
			}, timeout, interval).Should(Succeed())

			// check that an event was recorded for the reconciliation
			eventList := &eventsv1.EventList{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.List(ctx, eventList, client.InNamespace(Namespace))).To(Succeed())
				matchingEvents := filterEventsByReason(eventList.Items, "StageConfigurationReconciled")
				g.Expect(matchingEvents).ToNot(BeEmpty())
			}, timeout, interval).Should(Succeed())
		})
		It("should successfully update an existing stage with latest vector version ", func() {
			ctx := context.Background()
			vectorPortMock.EXPECT().GetLatestVectorVersion(gomock.Any(), VectorLatest).Return(VectorV101, nil)

			// create stageSync with v1.0.0 vector version
			CreateStageSync(ctx, k8sClient, StageSyncDev, Namespace, StageConfiguration, TargetNamespace, StageDev, VectorV100)

			var stageTemplate template.StageTemplate

			// check that the stageSync has been created and has valid properties
			stageSync := &global.StageSync{}
			stageSyncLookupKey := types.NamespacedName{Name: StageSyncDev, Namespace: TargetNamespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageSyncLookupKey, stageSync)).To(Succeed())
				g.Expect(stageSync.Name).To(Equal(StageSyncDev))
				g.Expect(json.Unmarshal(stageSync.Spec.StageTemplate.Raw, &stageTemplate)).To(Succeed())
				g.Expect(stageTemplate.Spec.Vector).To(Equal(VectorV100))
			}, timeout, interval).Should(Succeed())

			// create stageConfiguration
			CreateStageConfiguration(ctx, k8sClient, StageConfiguration, Namespace, TargetNamespace, StageDev, VectorLatest)

			// check that the stageConfiguration has been created and has valid properties
			stageConfiguration := &global.StageConfiguration{}
			stageConfigurationLookupKey := types.NamespacedName{Name: StageConfiguration, Namespace: Namespace}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageConfigurationLookupKey, stageConfiguration)).To(Succeed())
				g.Expect(stageConfiguration.Spec.Name).To(Equal(StageDev))
				g.Expect(stageConfiguration.Spec.Vector).To(Equal(VectorLatest))
				g.Expect(stageConfiguration.Spec.TargetNamespace).To(Equal(TargetNamespace))
			}, timeout, interval).Should(Succeed())

			// check that the stageSync has been updated with new vector version
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageSyncLookupKey, stageSync)).To(Succeed())
				g.Expect(json.Unmarshal(stageSync.Spec.StageTemplate.Raw, &stageTemplate)).To(Succeed())
				g.Expect(stageTemplate.Spec.Vector).To(Equal(VectorV101))
			}, timeout, interval).Should(Succeed())

			// check that the status condition is set to ready after reconcile
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, stageConfigurationLookupKey, stageConfiguration)).To(Succeed())
				g.Expect(stageConfiguration.Status.Conditions).To(HaveLen(1))
				g.Expect(stageConfiguration.Status.Conditions[0].Type).To(Equal(global.StageConfigurationReadyCondition))
				g.Expect(stageConfiguration.Status.Conditions[0].Status).To(Equal(metav1.ConditionTrue))
			}, timeout, interval).Should(Succeed())

			// check that an event was recorded for the reconciliation
			eventList := &eventsv1.EventList{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.List(ctx, eventList, client.InNamespace(Namespace))).To(Succeed())
				matchingEvents := filterEventsByReason(eventList.Items, "StageConfigurationReconciled")
				g.Expect(matchingEvents).ToNot(BeEmpty())
			}, timeout, interval).Should(Succeed())
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

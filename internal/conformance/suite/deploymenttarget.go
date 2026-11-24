package suite

import (
	"context"
	"reflect"
	"strings"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func registerDeploymentTargetScenarios(environment *Environment, setup *scenarioSetup) {
	ginkgo.Describe("DeploymentTarget", ginkgo.Label("DeploymentTarget"), func() {
		var target *konfidence.DeploymentTarget

		// Setup: the common scenario hook has already prepared the client,
		// namespace, credentials, and asynchronous assertion defaults.
		ginkgo.BeforeEach(func() {
			target = newDeploymentTarget(environment, setup.namespace)
		})

		ginkgo.Context("ownership and selection", func() {
			var foreignTargets []*konfidence.DeploymentTarget

			// Setup: prove the deployer is active before observing foreign targets.
			ginkgo.BeforeEach(func(ctx ginkgo.SpecContext) {
				createDeploymentTarget(ctx, environment.Client, target)
				expectTargetAccepted(ctx, environment.Client, target)
				foreignTargets = createForeignTargets(ctx, environment.Client, target)
			}, ginkgo.NodeTimeout(2*setup.timing.operation))

			// Tests: only the ownership guarantees are asserted here.
			ginkgo.It("[0018] reconciles only targets with a supported artifact type", ginkgo.Label("0018"),
				func(ctx ginkgo.SpecContext) {
					expectForeignTargetsUntouched(ctx, environment.Client, foreignTargets, false)
				}, setup.timing.observationSpecTimeout())

			ginkgo.It("[0019] does not update the status of foreign targets", ginkgo.Label("0019"),
				func(ctx ginkgo.SpecContext) {
					expectForeignTargetsUntouched(ctx, environment.Client, foreignTargets, true)
				}, setup.timing.observationSpecTimeout())
		})

		ginkgo.Context("with valid parameters", func() {
			// Setup.
			ginkgo.BeforeEach(func(ctx ginkgo.SpecContext) {
				createDeploymentTarget(ctx, environment.Client, target)
			}, ginkgo.NodeTimeout(setup.timing.operation))

			// Test.
			ginkgo.It("[0027] sets Ready=True after accepting the target", ginkgo.Label("0027"),
				func(ctx ginkgo.SpecContext) {
					expectTargetAccepted(ctx, environment.Client, target)
				}, setup.timing.statusSpecTimeout())
		})

		ginkgo.Context("with a missing parameter resource", func() {
			// Setup: change only the reference, leaving provider-owned payloads alone.
			ginkgo.BeforeEach(func(ctx ginkgo.SpecContext) {
				useMissingParameterResource(target)
				createDeploymentTarget(ctx, environment.Client, target)
			}, ginkgo.NodeTimeout(setup.timing.operation))

			// Tests.
			ginkgo.It("[0022] rejects invalid parameters", ginkgo.Label("0022"), func(ctx ginkgo.SpecContext) {
				gomega.Eventually(func(g gomega.Gomega) {
					current := readDeploymentTarget(ctx, g, environment.Client, target)
					g.Expect(meta.IsStatusConditionFalse(current.Status.Conditions, konfidence.DeploymentTargetReadyCondition)).
						To(gomega.BeTrue(), "missing parameter resource must result in Ready=False")
				}).WithContext(ctx).Should(gomega.Succeed())
			}, setup.timing.statusSpecTimeout())

			ginkgo.It("[0029] reports Ready=False with a reason and message", ginkgo.Label("0029"),
				func(ctx ginkgo.SpecContext) {
					gomega.Eventually(func(g gomega.Gomega) {
						current := readDeploymentTarget(ctx, g, environment.Client, target)
						condition := meta.FindStatusCondition(current.Status.Conditions, konfidence.DeploymentTargetReadyCondition)
						g.Expect(condition).NotTo(gomega.BeNil(), "Ready condition must be present")
						g.Expect(condition.Status).To(gomega.Equal(metav1.ConditionFalse))
						// Failed presence checks print only empty strings, not provider-owned messages.
						g.Expect(strings.TrimSpace(condition.Reason)).NotTo(gomega.BeEmpty(), "reason must not be empty")
						g.Expect(strings.TrimSpace(condition.Message)).NotTo(gomega.BeEmpty(), "message must not be empty")
					}).WithContext(ctx).Should(gomega.Succeed())
				}, setup.timing.statusSpecTimeout())
		})
	})
}

// Contract assertions shared by the keyword specs above.
func expectTargetAccepted(ctx context.Context, c client.Client, target *konfidence.DeploymentTarget) {
	ginkgo.GinkgoHelper()
	gomega.Eventually(func(g gomega.Gomega) {
		current := readDeploymentTarget(ctx, g, c, target)
		g.Expect(meta.IsStatusConditionTrue(current.Status.Conditions, konfidence.DeploymentTargetReadyCondition)).
			To(gomega.BeTrue(), "valid target must become Ready=True")
	}).WithContext(ctx).Should(gomega.Succeed())
}

func expectForeignTargetsUntouched(
	ctx context.Context, c client.Client, baselines []*konfidence.DeploymentTarget, statusOnly bool,
) {
	ginkgo.GinkgoHelper()
	gomega.Consistently(func(g gomega.Gomega) {
		for _, baseline := range baselines {
			current := readDeploymentTarget(ctx, g, c, baseline)
			unchanged := reflect.DeepEqual(baseline.Status, current.Status)
			if !statusOnly {
				unchanged = unchanged && reflect.DeepEqual(baseline.Spec, current.Spec) &&
					reflect.DeepEqual(baseline.Finalizers, current.Finalizers) &&
					reflect.DeepEqual(baseline.Labels, current.Labels) &&
					reflect.DeepEqual(baseline.Annotations, current.Annotations) &&
					reflect.DeepEqual(baseline.OwnerReferences, current.OwnerReferences) &&
					reflect.DeepEqual(baseline.DeletionTimestamp, current.DeletionTimestamp)
			}
			g.Expect(unchanged).To(gomega.BeTrue(), "foreign DeploymentTarget was modified")
		}
	}).WithContext(ctx).Should(gomega.Succeed())
}

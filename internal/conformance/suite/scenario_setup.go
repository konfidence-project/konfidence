package suite

import (
	"context"
	"fmt"
	"os"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultScenarioTimeout = time.Minute
	scenarioPollInterval   = 250 * time.Millisecond
	scenarioCleanupTimeout = 30 * time.Second
)

type scenarioTiming struct {
	operation   time.Duration
	observation time.Duration
}

func (timing scenarioTiming) statusSpecTimeout() ginkgo.SpecTimeout {
	// Allow one setup window and one status wait.
	return ginkgo.SpecTimeout(2 * timing.operation)
}

func (timing scenarioTiming) observationSpecTimeout() ginkgo.SpecTimeout {
	// Ownership checks additionally observe foreign targets for a full window.
	return ginkgo.SpecTimeout(2*timing.operation + timing.observation)
}

type scenarioSetup struct {
	namespace string
	timing    scenarioTiming
}

// registerScenarioSetup installs the common fixture lifecycle for all contract
// scenarios in the enclosing Describe. Resource-specific hooks run afterwards.
func registerScenarioSetup(environment *Environment) *scenarioSetup {
	timing, timingErr := loadScenarioTiming()
	setup := &scenarioSetup{timing: timing}
	ginkgo.BeforeEach(func(ctx ginkgo.SpecContext) {
		gomega.Expect(timingErr).To(gomega.Succeed())
		gomega.Expect(environment.Inputs).NotTo(gomega.BeNil(), "baseline inputs are required")
		gomega.SetDefaultEventuallyTimeout(timing.operation)
		gomega.SetDefaultEventuallyPollingInterval(scenarioPollInterval)
		// Gomega otherwise ignores its default Eventually timeout when a
		// SpecContext is supplied, relying only on the outer spec deadline.
		gomega.EnforceDefaultTimeoutsWhenUsingContexts()
		gomega.SetDefaultConsistentlyDuration(timing.observation)
		gomega.SetDefaultConsistentlyPollingInterval(scenarioPollInterval)
		initializeClient(environment)
		setup.namespace = createScenarioNamespace(ctx, environment.Client)
		createScenarioResources(ctx, environment, setup.namespace)
	}, ginkgo.NodeTimeout(timing.operation))
	return setup
}

func initializeClient(environment *Environment) {
	ginkgo.GinkgoHelper()
	if environment.Client != nil {
		gomega.Expect(corev1.AddToScheme(environment.Client.Scheme())).To(gomega.Succeed())
		gomega.Expect(konfidence.AddToScheme(environment.Client.Scheme())).To(gomega.Succeed())
		return
	}
	gomega.Expect(environment.Inputs.Cluster).NotTo(gomega.BeNil(), "cluster configuration is required")
	scheme := runtime.NewScheme()
	gomega.Expect(corev1.AddToScheme(scheme)).To(gomega.Succeed())
	gomega.Expect(konfidence.AddToScheme(scheme)).To(gomega.Succeed())
	c, err := client.New(environment.Inputs.Cluster, client.Options{Scheme: scheme})
	if err != nil {
		ginkgo.Fail("cannot initialize Kubernetes client")
	}
	environment.Client = c
}

func createScenarioNamespace(ctx context.Context, c client.Client) string {
	ginkgo.GinkgoHelper()
	namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{GenerateName: "kden-conformance-scenario-"}}
	if err := c.Create(ctx, namespace); err != nil {
		ginkgo.Fail("cannot create scenario namespace")
	}
	ginkgo.AddReportEntry("namespace", namespace.Name)
	// Register immediately so partial setup failures also clean up. Ginkgo gives
	// cleanup its own context, independent of the canceled/timed-out spec.
	ginkgo.DeferCleanup(func(cleanupCtx ginkgo.SpecContext) {
		if err := c.Delete(cleanupCtx, namespace); err != nil && !apierrors.IsNotFound(err) {
			ginkgo.Fail(fmt.Sprintf("cannot delete scenario namespace %q", namespace.Name))
		}
	}, ginkgo.NodeTimeout(scenarioCleanupTimeout))
	return namespace.Name
}

func createScenarioResources(ctx context.Context, environment *Environment, namespace string) {
	ginkgo.GinkgoHelper()
	for _, source := range environment.Inputs.TargetSetup {
		object := source.DeepCopy()
		object.SetNamespace(namespace)
		mapping, err := environment.Client.RESTMapper().
			RESTMapping(object.GroupVersionKind().GroupKind(), object.GroupVersionKind().Version)
		if err != nil || mapping.Scope.Name() != meta.RESTScopeNameNamespace {
			ginkgo.Fail("target setup resources must be discoverable and namespaced")
		}
		if err := environment.Client.Create(ctx, object); err != nil {
			// API response bodies can contain rejected credentials.
			ginkgo.Fail(fmt.Sprintf("cannot create target setup resource %q", object.GetName()))
		}
	}
}

func loadScenarioTiming() (scenarioTiming, error) {
	timing := scenarioTiming{operation: defaultScenarioTimeout, observation: 10 * time.Second}
	for name, value := range map[string]*time.Duration{
		"KDEN_CONFORMANCE_TIMEOUT":             &timing.operation,
		"KDEN_CONFORMANCE_OBSERVATION_TIMEOUT": &timing.observation,
	} {
		if raw := os.Getenv(name); raw != "" {
			duration, err := time.ParseDuration(raw)
			if err != nil || duration <= 0 {
				return timing, fmt.Errorf("%s must be a positive duration", name)
			}
			*value = duration
		}
	}
	return timing, nil
}

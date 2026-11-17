package stage

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/go-logr/logr"
	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	pkgctrl "github.com/konfidence-project/konfidence/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

func TestNamespaceExclusionsInRegisteredStageControllers(t *testing.T) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("requires envtest binaries")
	}
	environment := &envtest.Environment{CRDDirectoryPaths: []string{filepath.Join("..", "..", "test", "data", "crds")}, ErrorIfCRDPathMissing: true}
	config, err := environment.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := environment.Stop(); err != nil {
			t.Error(err)
		}
	}()
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := konfidence.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	c, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, ns := range []string{"fixtures", "allowed"} {
		if err := c.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns, Labels: map[string]string{pkgctrl.ProjectTypeLabel: "landscape"}}}); err != nil {
			t.Fatal(err)
		}
	}
	fixed := &konfidence.Stage{ObjectMeta: metav1.ObjectMeta{Name: "fixed", Namespace: "fixtures"}, Spec: konfidence.StageSpec{Vector: "dummy-vector"}}
	if err := c.Create(ctx, fixed); err != nil {
		t.Fatal(err)
	}
	fixed.Status = konfidence.StageStatus{Conditions: []metav1.Condition{{Type: "Ready", Status: metav1.ConditionFalse, Reason: "FixedSnapshot", Message: "Must survive reconciliation", LastTransitionTime: metav1.Now()}}}
	if err := c.Status().Update(ctx, fixed); err != nil {
		t.Fatal(err)
	}
	expectedStatus := fixed.Status
	version := &konfidence.StageVersion{
		ObjectMeta: metav1.ObjectMeta{Name: "fixed-version", Namespace: "fixtures"},
		Spec:       konfidence.StageVersionSpec{StageRef: &konfidence.StageReference{Name: "fixed"}, StageGeneration: 1, Vector: "dummy-vector"},
	}
	if err := c.Create(ctx, version); err != nil {
		t.Fatal(err)
	}
	version.Status.Conditions = expectedStatus.Conditions
	if err := c.Status().Update(ctx, version); err != nil {
		t.Fatal(err)
	}
	expectedVersionStatus := version.Status
	usage := &konfidence.StageVersionUsage{
		ObjectMeta: metav1.ObjectMeta{Name: "fixed-usage", Namespace: "fixtures"},
		Spec:       konfidence.StageVersionUsageSpec{StageVersionRef: &konfidence.StageVersionReference{Name: "fixed-version"}},
	}
	if err := c.Create(ctx, usage); err != nil {
		t.Fatal(err)
	}
	usage.Status.Conditions = expectedStatus.Conditions
	if err := c.Status().Update(ctx, usage); err != nil {
		t.Fatal(err)
	}
	expectedUsageStatus := usage.Status
	allowed := &konfidence.Stage{ObjectMeta: metav1.ObjectMeta{Name: "empty", Namespace: "allowed"}}
	if err := c.Create(ctx, allowed); err != nil {
		t.Fatal(err)
	}
	manager, err := ctrl.NewManager(config, ctrl.Options{Scheme: scheme, Metrics: metricsserver.Options{BindAddress: "0"}, HealthProbeBindAddress: "0"})
	if err != nil {
		t.Fatal(err)
	}
	policy, err := pkgctrl.ParseNamespaceExclusions("fixtures")
	if err != nil {
		t.Fatal(err)
	}
	if err := SetupControllers(manager, logr.Discard(), policy); err != nil {
		t.Fatal(err)
	}
	managerCtx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- manager.Start(managerCtx) }()
	defer func() {
		cancel()
		if err := <-done; err != nil {
			t.Error(err)
		}
	}()
	deadline := time.Now().Add(10 * time.Second)
	for {
		if err := c.Get(ctx, client.ObjectKeyFromObject(allowed), allowed); err != nil {
			t.Fatal(err)
		}
		if len(allowed.Status.Conditions) != 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("allowed controller did not run")
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err := c.Get(ctx, client.ObjectKeyFromObject(fixed), fixed); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expectedStatus, fixed.Status) {
		t.Fatal("registered controller changed excluded status")
	}
	versions := &konfidence.StageVersionList{}
	if err := c.List(ctx, versions, client.InNamespace("fixtures")); err != nil {
		t.Fatal(err)
	}
	if len(versions.Items) != 1 {
		t.Fatal("registered controller created excluded stage versions")
	}
	if err := c.Get(ctx, client.ObjectKeyFromObject(version), version); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expectedVersionStatus, version.Status) {
		t.Fatal("registered StageVersion controller changed excluded status")
	}
	if err := c.Get(ctx, client.ObjectKeyFromObject(usage), usage); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(expectedUsageStatus, usage.Status) {
		t.Fatal("registered StageVersionUsage controller changed excluded status")
	}
	deployments := &konfidence.VectorDeploymentList{}
	if err := c.List(ctx, deployments, client.InNamespace("fixtures")); err != nil {
		t.Fatal(err)
	}
	if len(deployments.Items) != 0 {
		t.Fatal("registered StageVersion controller created an excluded deployment")
	}
}

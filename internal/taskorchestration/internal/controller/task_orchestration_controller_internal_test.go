package controller

import (
	"context"
	"testing"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type noopRecorder struct{}

func (noopRecorder) Eventf(_ runtime.Object, _ runtime.Object, _, _, _, _ string, _ ...interface{}) {}

var _ events.EventRecorder = noopRecorder{}

// TestProcessTaskLayerIsIdempotentWithStaleCachedView asserts a task is created
// only once when the controller's cached view is stale (modeled by an empty map
// passed twice).
func TestProcessTaskLayerIsIdempotentWithStaleCachedView(t *testing.T) {
	ctx := context.Background()

	scheme := runtime.NewScheme()
	if err := konfidence.AddToScheme(scheme); err != nil {
		t.Fatalf("scheme: %v", err)
	}

	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	r := &TaskOrchestrationReconciler{Client: c, Scheme: scheme, Recorder: noopRecorder{}}

	vectorMigration := &konfidence.VectorMigration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "stage-version-stage-dev-migration",
			Namespace: "default",
			UID:       "843c08d3-9450-4802-88e8-00a848665c67",
		},
	}

	layer := []konfidence.TaskManifest{
		{Name: "task-0", Type: "k8s", Spec: runtime.RawExtension{Raw: []byte("{}")}},
	}

	// Empty map on both passes models a cache that never observed the first create.
	staleView := map[string]konfidence.TaskExecution{}
	succeeded := map[string]bool{}

	if _, err := r.processTaskLayer(ctx, vectorMigration, layer, staleView, succeeded); err != nil {
		t.Fatalf("first processTaskLayer: %v", err)
	}
	if _, err := r.processTaskLayer(ctx, vectorMigration, layer, staleView, succeeded); err != nil {
		t.Fatalf("second processTaskLayer: %v", err)
	}

	var taskExecutions konfidence.TaskExecutionList
	if err := c.List(ctx, &taskExecutions); err != nil {
		t.Fatalf("list TaskExecutions: %v", err)
	}

	countByTask := map[string]int{}
	for _, te := range taskExecutions.Items {
		countByTask[te.Spec.Name]++
	}

	if len(taskExecutions.Items) != 1 {
		t.Errorf("expected 1 TaskExecution, got %d", len(taskExecutions.Items))
	}
	if countByTask["task-0"] != 1 {
		t.Errorf("expected task-0 created once, got %d", countByTask["task-0"])
	}
}

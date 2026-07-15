package controller

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type noopRecorder struct{}

func (noopRecorder) Eventf(_ runtime.Object, _ runtime.Object, _, _, _, _ string, _ ...interface{}) {}

var _ events.EventRecorder = noopRecorder{}

func TestProcessTaskLayerIsIdempotentWithStaleCachedView(t *testing.T) {
	g := NewWithT(t)
	ctx := context.Background()

	scheme := runtime.NewScheme()
	g.Expect(konfidence.AddToScheme(scheme)).To(Succeed())

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

	_, err := r.processTaskLayer(ctx, vectorMigration, layer, staleView, succeeded)
	g.Expect(err).NotTo(HaveOccurred())

	_, err = r.processTaskLayer(ctx, vectorMigration, layer, staleView, succeeded)
	g.Expect(err).NotTo(HaveOccurred())

	var taskExecutions konfidence.TaskExecutionList
	g.Expect(c.List(ctx, &taskExecutions)).To(Succeed())

	g.Expect(taskExecutions.Items).To(HaveLen(1))
}

package controller

import (
	"context"
	"encoding/json"
	"testing"

	star "github.com/konfidence-project/konfidence/api/star/v1alpha1"
	pkgctrl "github.com/konfidence-project/konfidence/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// noopRecorder satisfies events.EventRecorder without buffering or formatting; suitable for unit tests where the
// recorded events are not asserted on.
type noopRecorder struct{}

func (noopRecorder) Eventf(_ runtime.Object, _ runtime.Object, _, _, _, _ string, _ ...interface{}) {
}

var _ events.EventRecorder = noopRecorder{}

// newReconcilerForVectorConfigTest constructs a reconciler backed by a controller-runtime fake client. The fake client
// is sufficient for handleVectorConfig because the function never touches subresources beyond the parent VD's status,
// and we only assert on top-level ConfigMap operations.
func newReconcilerForVectorConfigTest(t *testing.T, objects ...client.Object) (*VectorDeploymentReconciler, client.Client) {
	t.Helper()

	scheme := runtime.NewScheme()
	if err := star.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add star scheme: %v", err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add corev1 scheme: %v", err)
	}

	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objects...).
		WithStatusSubresource(&star.VectorDeployment{}).
		Build()

	return &VectorDeploymentReconciler{
		Client:   c,
		Scheme:   scheme,
		Recorder: noopRecorder{},
	}, c
}

func newTestVectorDeployment(name, configBlob string, results map[string]star.DeploymentResult) *star.VectorDeployment {
	const namespace = "landscape-a"
	vd := &star.VectorDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:       name,
			Namespace:  namespace,
			UID:        types.UID("uid-" + name),
			Generation: 1,
		},
		Spec: star.VectorDeploymentSpec{Vector: "https://example/vector:1.0.0"},
		Status: star.VectorDeploymentStatus{
			ResolvedVectorConfig: configBlob,
			DeploymentResults:    results,
		},
	}
	return vd
}

func TestHandleVectorConfig_EmptyState_WritesEmptyConfigMap(t *testing.T) {
	vd := newTestVectorDeployment("vd-empty", "", nil)
	r, c := newReconcilerForVectorConfigTest(t, vd)

	if err := r.handleVectorConfig(context.Background(), vd, logf.Log); err != nil {
		t.Fatalf("handleVectorConfig returned error: %v", err)
	}

	cm := &corev1.ConfigMap{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "vector-data-vd-empty", Namespace: "landscape-a"}, cm); err != nil {
		t.Fatalf("expected ConfigMap to exist, got: %v", err)
	}
	if cm.Immutable == nil || !*cm.Immutable {
		t.Errorf("expected Immutable=true, got: %v", cm.Immutable)
	}
	if cm.Labels[pkgctrl.ManagedByLabel] != VectorDeploymentControllerName {
		t.Errorf("expected managed-by label %q, got %q", VectorDeploymentControllerName, cm.Labels[pkgctrl.ManagedByLabel])
	}

	payload := map[string]json.RawMessage{}
	if err := json.Unmarshal([]byte(cm.Data[VectorConfigDataKey]), &payload); err != nil {
		t.Fatalf("payload not valid JSON: %v", err)
	}
	if string(payload["config"]) != "null" {
		t.Errorf("expected config to be null, got %q", payload["config"])
	}
	if string(payload["deploymentResults"]) != "{}" {
		t.Errorf("expected empty deploymentResults map, got %q", payload["deploymentResults"])
	}
	if !meta.IsStatusConditionTrue(vd.Status.Conditions, star.VectorConfigCommittedCondition) {
		t.Errorf("expected VectorConfigCommittedCondition=True")
	}
}

func TestHandleVectorConfig_AuthoredBlobIsForwardedVerbatim(t *testing.T) {
	authored := `{"featureFlags":{"darkMode":true}}`
	vd := newTestVectorDeployment("vd-authored", authored, nil)
	r, c := newReconcilerForVectorConfigTest(t, vd)

	if err := r.handleVectorConfig(context.Background(), vd, logf.Log); err != nil {
		t.Fatalf("handleVectorConfig returned error: %v", err)
	}

	cm := &corev1.ConfigMap{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "vector-data-vd-authored", Namespace: "landscape-a"}, cm); err != nil {
		t.Fatalf("expected ConfigMap to exist, got: %v", err)
	}

	payload := map[string]json.RawMessage{}
	if err := json.Unmarshal([]byte(cm.Data[VectorConfigDataKey]), &payload); err != nil {
		t.Fatalf("payload not valid JSON: %v", err)
	}
	if string(payload["config"]) != authored {
		t.Errorf("expected authored config to be forwarded verbatim, got %q", payload["config"])
	}
}

func TestHandleVectorConfig_RejectsInvalidAuthoredJSON(t *testing.T) {
	vd := newTestVectorDeployment("vd-bad", `not json {`, nil)
	r, _ := newReconcilerForVectorConfigTest(t, vd)

	err := r.handleVectorConfig(context.Background(), vd, logf.Log)
	if err == nil {
		t.Fatalf("expected error on invalid authored JSON, got nil")
	}
	if cond := meta.FindStatusCondition(vd.Status.Conditions, star.VectorConfigCommittedCondition); cond == nil || cond.Status != metav1.ConditionFalse {
		t.Errorf("expected VectorConfigCommittedCondition=False on bad payload, got: %#v", cond)
	}
}

func TestHandleVectorConfig_DeploymentResultsAreAggregated(t *testing.T) {
	results := map[string]star.DeploymentResult{
		"svc-a/result-1": {Name: "result-1", Type: "test", Spec: runtime.RawExtension{Raw: []byte(`{"endpoint":"http://a"}`)}},
		"svc-b/result-1": {Name: "result-1", Type: "test", Spec: runtime.RawExtension{Raw: []byte(`{"endpoint":"http://b"}`)}},
	}
	vd := newTestVectorDeployment("vd-results", "", results)
	r, c := newReconcilerForVectorConfigTest(t, vd)

	if err := r.handleVectorConfig(context.Background(), vd, logf.Log); err != nil {
		t.Fatalf("handleVectorConfig returned error: %v", err)
	}

	cm := &corev1.ConfigMap{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "vector-data-vd-results", Namespace: "landscape-a"}, cm); err != nil {
		t.Fatalf("expected ConfigMap to exist, got: %v", err)
	}

	payload := map[string]json.RawMessage{}
	if err := json.Unmarshal([]byte(cm.Data[VectorConfigDataKey]), &payload); err != nil {
		t.Fatalf("payload not valid JSON: %v", err)
	}
	got := map[string]star.DeploymentResult{}
	if err := json.Unmarshal(payload["deploymentResults"], &got); err != nil {
		t.Fatalf("deploymentResults not valid JSON: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 aggregated results, got %d", len(got))
	}
}

// TestHandleVectorConfig_NoOpWhenAlreadyPresent asserts the function trusts an existing ConfigMap and does not attempt
// to mutate it. Vector data is immutable per ADR-0024 -- once written, the controller has no business touching it.
func TestHandleVectorConfig_NoOpWhenAlreadyPresent(t *testing.T) {
	vd := newTestVectorDeployment("vd-existing", `{"x":1}`, nil)

	// Pre-populate a ConfigMap whose content differs from what handleVectorConfig would have written. The function must
	// honor it as-is rather than overwriting.
	preExisting := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vector-data-vd-existing",
			Namespace: "landscape-a",
			Labels: map[string]string{
				pkgctrl.ManagedByLabel: VectorDeploymentControllerName,
			},
		},
		Data: map[string]string{
			VectorConfigDataKey: `{"config":{"prior":true},"deploymentResults":{}}`,
		},
	}

	r, c := newReconcilerForVectorConfigTest(t, vd, preExisting)

	if err := r.handleVectorConfig(context.Background(), vd, logf.Log); err != nil {
		t.Fatalf("handleVectorConfig returned error: %v", err)
	}

	got := &corev1.ConfigMap{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "vector-data-vd-existing", Namespace: "landscape-a"}, got); err != nil {
		t.Fatalf("expected ConfigMap to still exist: %v", err)
	}
	if got.Data[VectorConfigDataKey] != preExisting.Data[VectorConfigDataKey] {
		t.Errorf("expected pre-existing payload to be preserved, got %q", got.Data[VectorConfigDataKey])
	}
	if !meta.IsStatusConditionTrue(vd.Status.Conditions, star.VectorConfigCommittedCondition) {
		t.Errorf("expected VectorConfigCommittedCondition=True even on no-op")
	}
}

// TestHandleVectorDeploymentDeletion_RemovesConfigMapAndFinalizer covers the teardown contract: when the VD is being
// deleted, the controller deletes the vector data ConfigMap explicitly (rather than relying solely on the K8s GC
// cascade) and then removes the VectorDataFinalizer so the API server can finalize the object.
func TestHandleVectorDeploymentDeletion_RemovesConfigMapAndFinalizer(t *testing.T) {
	now := metav1.Now()
	vd := newTestVectorDeployment("vd-tearing-down", "", nil)
	vd.Finalizers = []string{star.VectorDataFinalizer}
	vd.DeletionTimestamp = &now

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vector-data-vd-tearing-down",
			Namespace: "landscape-a",
		},
		Data: map[string]string{VectorConfigDataKey: `{"config":null,"deploymentResults":{}}`},
	}

	r, c := newReconcilerForVectorConfigTest(t, vd, cm)
	ctx := context.Background()

	if _, err := r.handleVectorDeploymentDeletion(ctx, vd, logf.Log); err != nil {
		t.Fatalf("handleVectorDeploymentDeletion: %v", err)
	}

	got := &corev1.ConfigMap{}
	err := c.Get(ctx, types.NamespacedName{Name: "vector-data-vd-tearing-down", Namespace: "landscape-a"}, got)
	if err == nil {
		t.Fatalf("expected ConfigMap to be deleted, but it is still present")
	}
	if !apierrors.IsNotFound(err) {
		t.Fatalf("unexpected error fetching deleted ConfigMap: %v", err)
	}

	// The fake client removes the VD object once the last finalizer is patched away (DeletionTimestamp is set), so we
	// inspect the in-memory copy that handleVectorDeploymentDeletion mutated rather than re-reading from the cache.
	for _, f := range vd.Finalizers {
		if f == star.VectorDataFinalizer {
			t.Errorf("expected VectorDataFinalizer to be removed, still present in %v", vd.Finalizers)
		}
	}
}

// TestHandleVectorDeploymentDeletion_NoOpWhenConfigMapAlreadyGone exercises the idempotent path: a previous reconcile
// (or the K8s GC cascade) may have already removed the ConfigMap. The handler must still drop the finalizer cleanly.
func TestHandleVectorDeploymentDeletion_NoOpWhenConfigMapAlreadyGone(t *testing.T) {
	now := metav1.Now()
	vd := newTestVectorDeployment("vd-already-gone", "", nil)
	vd.Finalizers = []string{star.VectorDataFinalizer}
	vd.DeletionTimestamp = &now

	r, _ := newReconcilerForVectorConfigTest(t, vd) // no ConfigMap pre-populated
	ctx := context.Background()

	if _, err := r.handleVectorDeploymentDeletion(ctx, vd, logf.Log); err != nil {
		t.Fatalf("handleVectorDeploymentDeletion: %v", err)
	}

	for _, f := range vd.Finalizers {
		if f == star.VectorDataFinalizer {
			t.Errorf("expected VectorDataFinalizer to be removed even when ConfigMap was already gone")
		}
	}
}

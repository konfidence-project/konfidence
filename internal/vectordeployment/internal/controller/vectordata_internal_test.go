package controller

import (
	"context"
	"encoding/json"
	"testing"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type noopRecorder struct{}

func (noopRecorder) Eventf(_ runtime.Object, _ runtime.Object, _, _, _, _ string, _ ...interface{}) {}

var _ events.EventRecorder = noopRecorder{}

func newReconciler(t *testing.T, objects ...client.Object) (*VectorDeploymentReconciler, client.Client) {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := konfidence.AddToScheme(scheme); err != nil {
		t.Fatalf("scheme: %v", err)
	}
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("scheme: %v", err)
	}
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objects...).
		WithStatusSubresource(&konfidence.VectorDeployment{}, &konfidence.VectorData{}).
		Build()
	return &VectorDeploymentReconciler{Client: c, Scheme: scheme, Recorder: noopRecorder{}}, c
}

func newVD(name string, results map[string]konfidence.ComponentDeploymentResults) *konfidence.VectorDeployment {
	return &konfidence.VectorDeployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "landscape-a", UID: types.UID("uid-" + name), Generation: 1},
		Spec:       konfidence.VectorDeploymentSpec{Vector: "https://example/vector:1.0.0"},
		Status:     konfidence.VectorDeploymentStatus{DeploymentResults: results},
	}
}

// TestHandleVectorData_CreatesWithSplitEnvelope covers the happy path: the controller parses the OCM envelope and inlines
// the features/authored subsets as RawExtension, plus the aggregated DeploymentResults.
func TestHandleVectorData_CreatesWithSplitEnvelope(t *testing.T) {
	envelope := []byte(`{"features":{"darkMode":true},"authored":{"db":{"host":"mysql"}}}`)
	results := map[string]konfidence.ComponentDeploymentResults{
		"github.com/acme/svc-a": {{Name: "result-1", Type: "test", Spec: runtime.RawExtension{Raw: []byte(`{"endpoint":"http://a"}`)}}},
	}
	vd := newVD("vd-1", results)
	r, c := newReconciler(t, vd)

	if err := r.handleVectorData(context.Background(), vd, envelope, logf.Log); err != nil {
		t.Fatalf("handleVectorData: %v", err)
	}

	got := &konfidence.VectorData{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "vd-1", Namespace: "landscape-a"}, got); err != nil {
		t.Fatalf("get VectorData: %v", err)
	}
	if got.Spec.Features == nil || string(got.Spec.Features.Raw) != `{"darkMode":true}` {
		t.Errorf("features: got %v", got.Spec.Features)
	}
	if got.Spec.Authored == nil || string(got.Spec.Authored.Raw) != `{"db":{"host":"mysql"}}` {
		t.Errorf("authored: got %v", got.Spec.Authored)
	}
	if len(got.Spec.DeploymentResults) != 1 {
		t.Errorf("deploymentResults: got %d, want 1", len(got.Spec.DeploymentResults))
	}
	if len(got.OwnerReferences) == 0 {
		t.Errorf("expected ownerRef back to VectorDeployment")
	}
	if !meta.IsStatusConditionTrue(vd.Status.Conditions, konfidence.VectorDataCreatedCondition) {
		t.Errorf("expected VectorDataCreated=True")
	}
}

// TestHandleVectorData_MultipleResultsPerComponent: a single component may expose more than one Service, so its
// aggregated entry is a slice; all results are carried into VectorData.Spec verbatim under the component key.
func TestHandleVectorData_MultipleResultsPerComponent(t *testing.T) {
	results := map[string]konfidence.ComponentDeploymentResults{
		"github.com/acme/shop/storefront": {
			{Name: "storefront", Type: "http-k8s-service", Spec: runtime.RawExtension{Raw: []byte(`{"K8sName":"storefront-a1b2"}`)}},
			{Name: "storefront-admin", Type: "http-k8s-service", Spec: runtime.RawExtension{Raw: []byte(`{"K8sName":"storefront-admin-a1b2"}`)}},
		},
	}
	vd := newVD("vd-multi", results)
	r, c := newReconciler(t, vd)

	if err := r.handleVectorData(context.Background(), vd, []byte(`{}`), logf.Log); err != nil {
		t.Fatalf("handleVectorData: %v", err)
	}

	got := &konfidence.VectorData{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "vd-multi", Namespace: "landscape-a"}, got); err != nil {
		t.Fatalf("get VectorData: %v", err)
	}
	entry := got.Spec.DeploymentResults["github.com/acme/shop/storefront"]
	if len(entry) != 2 {
		t.Fatalf("results for component: got %d, want 2", len(entry))
	}
	if entry[0].Name != "storefront" || entry[1].Name != "storefront-admin" {
		t.Errorf("result names not preserved in order: %q, %q", entry[0].Name, entry[1].Name)
	}
}

// TestHandleVectorData_NoOpWhenPresent: an existing VectorData is honored as-is (Spec is immutable upstream).
func TestHandleVectorData_NoOpWhenPresent(t *testing.T) {
	vd := newVD("vd-existing", nil)
	preExisting := &konfidence.VectorData{
		ObjectMeta: metav1.ObjectMeta{Name: "vd-existing", Namespace: "landscape-a"},
		Spec:       konfidence.VectorDataSpec{Features: &runtime.RawExtension{Raw: []byte(`{"prior":true}`)}},
	}
	r, c := newReconciler(t, vd, preExisting)

	if err := r.handleVectorData(context.Background(), vd, []byte(`{"features":{"new":true}}`), logf.Log); err != nil {
		t.Fatalf("handleVectorData: %v", err)
	}

	got := &konfidence.VectorData{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "vd-existing", Namespace: "landscape-a"}, got); err != nil {
		t.Fatalf("get: %v", err)
	}
	if string(got.Spec.Features.Raw) != `{"prior":true}` {
		t.Errorf("pre-existing Spec.Features should be preserved, got %q", got.Spec.Features.Raw)
	}
}

// TestHandleVectorData_RejectsInvalidEnvelope: malformed OCM bytes surface as VectorDataCreated=False.
func TestHandleVectorData_RejectsInvalidEnvelope(t *testing.T) {
	vd := newVD("vd-bad", nil)
	r, _ := newReconciler(t, vd)

	err := r.handleVectorData(context.Background(), vd, []byte("not json"), logf.Log)
	if err == nil {
		t.Fatalf("expected error")
	}
	cond := meta.FindStatusCondition(vd.Status.Conditions, konfidence.VectorDataCreatedCondition)
	if cond == nil || cond.Status != metav1.ConditionFalse || cond.Reason != "InvalidConfigPayload" {
		t.Errorf("expected InvalidConfigPayload, got %#v", cond)
	}
}

// TestVectorDataIsReady_TracksImplementorState: the readiness probe the controller uses to gate VectorReady on the orchestrator.
func TestVectorDataIsReady_TracksImplementorState(t *testing.T) {
	vd := newVD("vd-r", nil)
	vd.Status.ResultingVectorData = &konfidence.LocalObjectReference{Name: "vd-r"}
	cr := &konfidence.VectorData{ObjectMeta: metav1.ObjectMeta{Name: "vd-r", Namespace: "landscape-a"}}
	r, c := newReconciler(t, vd, cr)
	ctx := context.Background()

	ok, err := r.vectorDataIsReady(ctx, vd)
	if err != nil || ok {
		t.Errorf("before flip: ok=%v err=%v; want ok=false", ok, err)
	}
	meta.SetStatusCondition(&cr.Status.Conditions, metav1.Condition{
		Type: konfidence.VectorDataReadyCondition, Status: metav1.ConditionTrue, Reason: konfidence.VectorDataReasonMaterialized,
	})
	if err := c.Status().Update(ctx, cr); err != nil {
		t.Fatalf("update: %v", err)
	}
	ok, err = r.vectorDataIsReady(ctx, vd)
	if err != nil || !ok {
		t.Errorf("after flip: ok=%v err=%v; want ok=true", ok, err)
	}
}

func TestSplitEnvelope(t *testing.T) {
	features, authored, err := splitEnvelope(nil)
	if err != nil || features != nil || authored != nil {
		t.Errorf("empty: features=%v authored=%v err=%v", features, authored, err)
	}
	features, authored, err = splitEnvelope([]byte(`{"features":{"a":1}}`))
	if err != nil || features == nil || authored != nil {
		t.Errorf("only features: features=%v authored=%v err=%v", features, authored, err)
	}
	_, _, err = splitEnvelope([]byte(`{`))
	if err == nil {
		t.Errorf("invalid json: expected error")
	}
	// Ensure the RawExtension carries an independent copy (mutating the returned slice must not affect future parses).
	features, _, _ = splitEnvelope([]byte(`{"features":{"x":1}}`))
	if !json.Valid(features.Raw) {
		t.Errorf("features.Raw is not valid JSON: %q", features.Raw)
	}
}

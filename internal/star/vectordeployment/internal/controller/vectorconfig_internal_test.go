package controller

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
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
	"ocm.software/open-component-model/bindings/go/oci/compref"
	ociv1 "ocm.software/open-component-model/bindings/go/oci/spec/repository/v1/oci"
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

// stubVectorOcmPort is a tiny hand-rolled stub for white-box tests in this same package. We can't reuse the
// gomock-generated MockVectorOcmPort here because it lives in a sub-package that imports `package controller`,
// which would form a test-time import cycle. Production tests in `controller_test` (a different package) keep using
// the generated mock.
type stubVectorOcmPort struct {
	getVectorDescriptor func(ctx context.Context, ref compref.Ref) (VectorDescriptor, error)
}

func (s stubVectorOcmPort) GetVectorDescriptor(ctx context.Context, ref compref.Ref) (VectorDescriptor, error) {
	if s.getVectorDescriptor == nil {
		return VectorDescriptor{}, errors.New("stubVectorOcmPort.GetVectorDescriptor not configured")
	}
	return s.getVectorDescriptor(ctx, ref)
}

func (s stubVectorOcmPort) GetArtifactManifestByReference(_ context.Context, _ compref.Ref) (ArtifactManifest, error) {
	return ArtifactManifest{}, errors.New("stubVectorOcmPort.GetArtifactManifestByReference not used by these tests")
}

var _ VectorOcmPort = stubVectorOcmPort{}

// testVectorRef is the OCM reference passed into handleVectorConfig in unit tests. Its only role is being available
// for the lazy re-fetch path; tests that exercise that path inject a stub OCM port that returns the expected blob.
var testVectorRef = compref.Ref{
	Repository: &ociv1.Repository{BaseUrl: "https://example.test"},
	Component:  "github.com/example/test/vector",
	Version:    "v0.0.1",
}

// newReconcilerForVectorConfigTest constructs a reconciler backed by a controller-runtime fake client. The fake client
// is sufficient for handleVectorConfig because the function never touches subresources beyond the parent VD's status,
// and we only assert on top-level ConfigMap operations.
func newReconcilerForVectorConfigTest(t *testing.T, ocmAdapter VectorOcmPort, objects ...client.Object) (*VectorDeploymentReconciler, client.Client) {
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
		Client:     c,
		Scheme:     scheme,
		Recorder:   noopRecorder{},
		OcmAdapter: ocmAdapter,
	}, c
}

func newTestVectorDeployment(name string, results map[string]star.DeploymentResult) *star.VectorDeployment {
	const namespace = "landscape-a"
	return &star.VectorDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:       name,
			Namespace:  namespace,
			UID:        types.UID("uid-" + name),
			Generation: 1,
		},
		Spec: star.VectorDeploymentSpec{Vector: "https://example/vector:1.0.0"},
		Status: star.VectorDeploymentStatus{
			DeploymentResults: results,
		},
	}
}

func decodePayload(t *testing.T, cm *corev1.ConfigMap) (config json.RawMessage, results map[string]star.DeploymentResult) {
	t.Helper()
	if err := json.Unmarshal([]byte(cm.Data[VectorConfigDataKey]), &config); err != nil {
		t.Fatalf("config.json is not valid JSON: %v", err)
	}
	if err := json.Unmarshal([]byte(cm.Data[VectorDeploymentResultsDataKey]), &results); err != nil {
		t.Fatalf("deployment-results.json is not valid JSON: %v", err)
	}
	return
}

func TestHandleVectorConfig_EmptyState_WritesEmptyConfigMap(t *testing.T) {
	vd := newTestVectorDeployment("vd-empty", nil)
	r, c := newReconcilerForVectorConfigTest(t, nil, vd)

	if err := r.handleVectorConfig(context.Background(), vd, testVectorRef, nil, logf.Log); err != nil {
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

	if got := cm.Data[VectorConfigDataKey]; got != "null" {
		t.Errorf("expected config.json to be 'null', got %q", got)
	}
	if got := cm.Data[VectorDeploymentResultsDataKey]; got != "{}" {
		t.Errorf("expected deployment-results.json to be '{}', got %q", got)
	}
	if !meta.IsStatusConditionTrue(vd.Status.Conditions, star.VectorConfigCommittedCondition) {
		t.Errorf("expected VectorConfigCommittedCondition=True")
	}
	if vd.Status.ResolvedVectorConfigHash != "" {
		t.Errorf("expected empty ResolvedVectorConfigHash when there is no authored config, got %q",
			vd.Status.ResolvedVectorConfigHash)
	}
}

func TestHandleVectorConfig_AuthoredBlobIsForwardedVerbatimAndHashed(t *testing.T) {
	authored := []byte(`{"featureFlags":{"darkMode":true}}`)
	vd := newTestVectorDeployment("vd-authored", nil)
	r, c := newReconcilerForVectorConfigTest(t, nil, vd)

	if err := r.handleVectorConfig(context.Background(), vd, testVectorRef, authored, logf.Log); err != nil {
		t.Fatalf("handleVectorConfig returned error: %v", err)
	}

	cm := &corev1.ConfigMap{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "vector-data-vd-authored", Namespace: "landscape-a"}, cm); err != nil {
		t.Fatalf("expected ConfigMap to exist, got: %v", err)
	}

	if got := cm.Data[VectorConfigDataKey]; got != string(authored) {
		t.Errorf("expected authored config to be forwarded verbatim, got %q", got)
	}
	if got := cm.Data[VectorDeploymentResultsDataKey]; got != "{}" {
		t.Errorf("expected deployment-results.json to be '{}' when no results exist, got %q", got)
	}

	wantHash := sha256Hex(authored)
	if vd.Status.ResolvedVectorConfigHash != wantHash {
		t.Errorf("expected ResolvedVectorConfigHash %q, got %q", wantHash, vd.Status.ResolvedVectorConfigHash)
	}
}

func TestHandleVectorConfig_RejectsInvalidAuthoredJSON(t *testing.T) {
	vd := newTestVectorDeployment("vd-bad", nil)
	r, _ := newReconcilerForVectorConfigTest(t, nil, vd)

	err := r.handleVectorConfig(context.Background(), vd, testVectorRef, []byte("not json {"), logf.Log)
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
	vd := newTestVectorDeployment("vd-results", results)
	r, c := newReconcilerForVectorConfigTest(t, nil, vd)

	if err := r.handleVectorConfig(context.Background(), vd, testVectorRef, nil, logf.Log); err != nil {
		t.Fatalf("handleVectorConfig returned error: %v", err)
	}

	cm := &corev1.ConfigMap{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "vector-data-vd-results", Namespace: "landscape-a"}, cm); err != nil {
		t.Fatalf("expected ConfigMap to exist, got: %v", err)
	}
	_, got := decodePayload(t, cm)
	if len(got) != 2 {
		t.Errorf("expected 2 aggregated results, got %d", len(got))
	}
}

// TestHandleVectorConfig_NoOpWhenAlreadyPresent asserts the function trusts an existing ConfigMap and does not attempt
// to mutate it. Vector data is immutable per ADR-0024 -- once written, the controller has no business touching it.
func TestHandleVectorConfig_NoOpWhenAlreadyPresent(t *testing.T) {
	vd := newTestVectorDeployment("vd-existing", nil)

	preExistingConfig := `{"prior":true}`
	preExisting := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vector-data-vd-existing",
			Namespace: "landscape-a",
			Labels: map[string]string{
				pkgctrl.ManagedByLabel: VectorDeploymentControllerName,
			},
		},
		Data: map[string]string{
			VectorConfigDataKey:            preExistingConfig,
			VectorDeploymentResultsDataKey: "{}",
		},
	}

	r, c := newReconcilerForVectorConfigTest(t, nil, vd, preExisting)

	if err := r.handleVectorConfig(context.Background(), vd, testVectorRef, []byte(`{"would-overwrite":true}`), logf.Log); err != nil {
		t.Fatalf("handleVectorConfig returned error: %v", err)
	}

	got := &corev1.ConfigMap{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "vector-data-vd-existing", Namespace: "landscape-a"}, got); err != nil {
		t.Fatalf("expected ConfigMap to still exist: %v", err)
	}
	if got.Data[VectorConfigDataKey] != preExistingConfig {
		t.Errorf("expected pre-existing config.json to be preserved, got %q", got.Data[VectorConfigDataKey])
	}
	if !meta.IsStatusConditionTrue(vd.Status.Conditions, star.VectorConfigCommittedCondition) {
		t.Errorf("expected VectorConfigCommittedCondition=True even on no-op")
	}
}

// TestHandleVectorConfig_RecreatesAndRefetchesWhenConfigMapMissing exercises the lazy re-fetch path: on a later
// reconcile (where the in-memory blob from the initial OCM fetch is no longer in scope) the handler must pull the
// blob again via the OCM adapter and use it to recreate the missing ConfigMap. Detection for "later reconcile" is
// `vd.Status.ResolvedVectorConfigHash != ""`, so we pre-populate it.
func TestHandleVectorConfig_RecreatesAndRefetchesWhenConfigMapMissing(t *testing.T) {
	expectedBlob := []byte(`{"a":1}`)

	vd := newTestVectorDeployment("vd-missing", nil)
	vd.Status.ResolvedVectorConfigHash = sha256Hex(expectedBlob)

	calls := 0
	stub := stubVectorOcmPort{
		getVectorDescriptor: func(_ context.Context, ref compref.Ref) (VectorDescriptor, error) {
			calls++
			if ref != testVectorRef {
				t.Errorf("unexpected ref: %+v", ref)
			}
			return VectorDescriptor{Configuration: expectedBlob}, nil
		},
	}

	r, c := newReconcilerForVectorConfigTest(t, stub, vd)
	ctx := context.Background()

	if err := r.handleVectorConfig(ctx, vd, testVectorRef, nil, logf.Log); err != nil {
		t.Fatalf("handleVectorConfig: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected exactly one OCM re-fetch, got %d", calls)
	}

	cm := &corev1.ConfigMap{}
	if err := c.Get(ctx, types.NamespacedName{Name: "vector-data-vd-missing", Namespace: "landscape-a"}, cm); err != nil {
		if apierrors.IsNotFound(err) {
			t.Fatalf("ConfigMap should have been created when missing on entry")
		}
		t.Fatalf("unexpected get error: %v", err)
	}
	if got := cm.Data[VectorConfigDataKey]; got != string(expectedBlob) {
		t.Errorf("expected re-fetched blob to be written verbatim, got %q", got)
	}
}

// TestHandleVectorConfig_LazyRefetchPropagatesError covers the error path of the lazy re-fetch: when the OCM adapter
// fails the handler must surface the error and flip VectorConfigCommitted to False with a recognizable reason.
func TestHandleVectorConfig_LazyRefetchPropagatesError(t *testing.T) {
	vd := newTestVectorDeployment("vd-refetch-fail", nil)
	vd.Status.ResolvedVectorConfigHash = "non-empty-marker"

	stub := stubVectorOcmPort{
		getVectorDescriptor: func(_ context.Context, _ compref.Ref) (VectorDescriptor, error) {
			return VectorDescriptor{}, errors.New("registry unavailable")
		},
	}
	r, _ := newReconcilerForVectorConfigTest(t, stub, vd)

	err := r.handleVectorConfig(context.Background(), vd, testVectorRef, nil, logf.Log)
	if err == nil {
		t.Fatalf("expected error from lazy re-fetch, got nil")
	}
	if !strings.Contains(err.Error(), "registry unavailable") {
		t.Errorf("expected wrapped adapter error, got: %v", err)
	}
	cond := meta.FindStatusCondition(vd.Status.Conditions, star.VectorConfigCommittedCondition)
	if cond == nil || cond.Status != metav1.ConditionFalse || cond.Reason != "ConfigBlobRefetchFailed" {
		t.Errorf("expected VectorConfigCommitted=False/ConfigBlobRefetchFailed, got: %#v", cond)
	}
}

// TestHandleVectorDeploymentDeletion_RemovesConfigMapAndFinalizer covers the teardown contract: when the VD is being
// deleted, the controller deletes the vector data ConfigMap explicitly (rather than relying solely on the K8s GC
// cascade) and then removes the VectorDataFinalizer so the API server can finalize the object.
func TestHandleVectorDeploymentDeletion_RemovesConfigMapAndFinalizer(t *testing.T) {
	now := metav1.Now()
	vd := newTestVectorDeployment("vd-tearing-down", nil)
	vd.Finalizers = []string{star.VectorDataFinalizer}
	vd.DeletionTimestamp = &now

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vector-data-vd-tearing-down",
			Namespace: "landscape-a",
		},
		Data: map[string]string{
			VectorConfigDataKey:            "null",
			VectorDeploymentResultsDataKey: "{}",
		},
	}

	r, c := newReconcilerForVectorConfigTest(t, nil, vd, cm)
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
	vd := newTestVectorDeployment("vd-already-gone", nil)
	vd.Finalizers = []string{star.VectorDataFinalizer}
	vd.DeletionTimestamp = &now

	r, _ := newReconcilerForVectorConfigTest(t, nil, vd)
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

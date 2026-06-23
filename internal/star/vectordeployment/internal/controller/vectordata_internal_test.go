package controller

import (
	"context"
	"errors"
	"strings"
	"testing"

	star "github.com/konfidence-project/konfidence/api/star/v1alpha1"
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

// testVectorRef is the OCM reference passed into handleVectorData in unit tests. Its only role is being available
// for the lazy re-fetch path; tests that exercise that path inject a stub OCM port that returns the expected blob.
var testVectorRef = compref.Ref{
	Repository: &ociv1.Repository{BaseUrl: "https://example.test"},
	Component:  "github.com/example/test/vector",
	Version:    "v0.0.1",
}

func newReconcilerForVectorDataTest(t *testing.T, ocmAdapter VectorOcmPort, objects ...client.Object) (*VectorDeploymentReconciler, client.Client) {
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
		WithStatusSubresource(&star.VectorDeployment{}, &star.VectorData{}).
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

// TestHandleVectorData_CreatesVectorDataWithInlinedPayload covers the happy path: the first reconcile passes the
// just-fetched OCM blob through `freshConfig`, and the controller emits a VectorData CR whose Spec carries the
// authored bytes verbatim plus the aggregated DeploymentResults. The runtime-specific implementor (not exercised
// here) is expected to subsequently flip VectorData.Status.Ready.
func TestHandleVectorData_CreatesVectorDataWithInlinedPayload(t *testing.T) {
	authored := []byte(`{"featureFlags":{"darkMode":true}}`)
	results := map[string]star.DeploymentResult{
		"svc-a/result-1": {Name: "result-1", Type: "test", Spec: runtime.RawExtension{Raw: []byte(`{"endpoint":"http://a"}`)}},
	}
	vd := newTestVectorDeployment("vd-create", results)
	r, c := newReconcilerForVectorDataTest(t, nil, vd)

	if err := r.handleVectorData(context.Background(), vd, testVectorRef, authored, logf.Log); err != nil {
		t.Fatalf("handleVectorData returned error: %v", err)
	}

	got := &star.VectorData{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "vd-create", Namespace: "landscape-a"}, got); err != nil {
		t.Fatalf("expected VectorData to exist, got: %v", err)
	}
	if string(got.Spec.Config) != string(authored) {
		t.Errorf("expected Spec.Config to be the authored blob verbatim, got %q", got.Spec.Config)
	}
	if len(got.Spec.DeploymentResults) != 1 {
		t.Errorf("expected 1 deployment result inlined, got %d", len(got.Spec.DeploymentResults))
	}
	if vd.Status.ResultingVectorData == nil || vd.Status.ResultingVectorData.Name != "vd-create" {
		t.Errorf("expected Status.ResultingVectorData to point at the created VectorData, got %#v", vd.Status.ResultingVectorData)
	}
	if !meta.IsStatusConditionTrue(vd.Status.Conditions, star.VectorDataCreatedCondition) {
		t.Errorf("expected VectorDataCreatedCondition=True")
	}

	// Owner reference points back to the VD so deletion cascades.
	if len(got.OwnerReferences) == 0 {
		t.Errorf("expected VectorData to have an owner reference back to the VectorDeployment")
	}
}

// TestHandleVectorData_OmitsConfigWhenAuthoredBlobAbsent covers the case where the vector did not declare a
// vector-config resource. The VectorData.Spec.Config should remain empty (nil), and the controller must still
// progress.
func TestHandleVectorData_OmitsConfigWhenAuthoredBlobAbsent(t *testing.T) {
	vd := newTestVectorDeployment("vd-no-config", nil)
	r, c := newReconcilerForVectorDataTest(t, nil, vd)

	if err := r.handleVectorData(context.Background(), vd, testVectorRef, nil, logf.Log); err != nil {
		t.Fatalf("handleVectorData returned error: %v", err)
	}

	got := &star.VectorData{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "vd-no-config", Namespace: "landscape-a"}, got); err != nil {
		t.Fatalf("expected VectorData to exist, got: %v", err)
	}
	if len(got.Spec.Config) != 0 {
		t.Errorf("expected Spec.Config to be empty when no authored blob, got %q", got.Spec.Config)
	}
	if len(got.Spec.DeploymentResults) != 0 {
		t.Errorf("expected Spec.DeploymentResults to be empty, got %v", got.Spec.DeploymentResults)
	}
}

// TestHandleVectorData_RejectsInvalidAuthoredJSON ensures the Star side surfaces malformed OCM payloads explicitly
// rather than emitting a VectorData that the implementor would fail to process.
func TestHandleVectorData_RejectsInvalidAuthoredJSON(t *testing.T) {
	vd := newTestVectorDeployment("vd-bad", nil)
	r, _ := newReconcilerForVectorDataTest(t, nil, vd)

	err := r.handleVectorData(context.Background(), vd, testVectorRef, []byte("not json {"), logf.Log)
	if err == nil {
		t.Fatalf("expected error on invalid authored JSON, got nil")
	}
	cond := meta.FindStatusCondition(vd.Status.Conditions, star.VectorDataCreatedCondition)
	if cond == nil || cond.Status != metav1.ConditionFalse || cond.Reason != "InvalidConfigPayload" {
		t.Errorf("expected VectorDataCreated=False/InvalidConfigPayload, got: %#v", cond)
	}
}

// TestHandleVectorData_NoOpWhenAlreadyPresent asserts the function trusts an existing VectorData and does not mutate
// it. VectorData.Spec is immutable upstream so any pre-existing CR was written by us.
func TestHandleVectorData_NoOpWhenAlreadyPresent(t *testing.T) {
	vd := newTestVectorDeployment("vd-existing", nil)
	preExisting := &star.VectorData{
		ObjectMeta: metav1.ObjectMeta{Name: "vd-existing", Namespace: "landscape-a"},
		Spec:       star.VectorDataSpec{Config: []byte(`{"prior":true}`)},
	}
	r, c := newReconcilerForVectorDataTest(t, nil, vd, preExisting)

	if err := r.handleVectorData(context.Background(), vd, testVectorRef, []byte(`{"would-overwrite":true}`), logf.Log); err != nil {
		t.Fatalf("handleVectorData returned error: %v", err)
	}

	got := &star.VectorData{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "vd-existing", Namespace: "landscape-a"}, got); err != nil {
		t.Fatalf("expected VectorData to still exist: %v", err)
	}
	if string(got.Spec.Config) != `{"prior":true}` {
		t.Errorf("expected pre-existing Spec.Config to be preserved, got %q", got.Spec.Config)
	}
	if vd.Status.ResultingVectorData == nil || vd.Status.ResultingVectorData.Name != "vd-existing" {
		t.Errorf("expected Status.ResultingVectorData to reference the existing VectorData")
	}
}

// TestHandleVectorData_RefetchesWhenVectorDataMissing covers the lazy re-fetch path. On a later reconcile (where
// freshConfig is nil), the handler must pull the blob via the OCM adapter and use it to (re)create the missing
// VectorData. We mark "later reconcile" by pre-populating Status.ResolvedVectorOcm.
func TestHandleVectorData_RefetchesWhenVectorDataMissing(t *testing.T) {
	expectedBlob := []byte(`{"a":1}`)

	vd := newTestVectorDeployment("vd-missing", nil)
	vd.Status.ResolvedVectorOcm = `{"meta":{"schemaVersion":"v2"},"component":{}}` // any non-empty marker

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
	r, c := newReconcilerForVectorDataTest(t, stub, vd)
	ctx := context.Background()

	if err := r.handleVectorData(ctx, vd, testVectorRef, nil, logf.Log); err != nil {
		t.Fatalf("handleVectorData: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected exactly one OCM re-fetch, got %d", calls)
	}
	got := &star.VectorData{}
	if err := c.Get(ctx, types.NamespacedName{Name: "vd-missing", Namespace: "landscape-a"}, got); err != nil {
		if apierrors.IsNotFound(err) {
			t.Fatalf("VectorData should have been created when missing on entry")
		}
		t.Fatalf("unexpected get error: %v", err)
	}
	if string(got.Spec.Config) != string(expectedBlob) {
		t.Errorf("expected re-fetched blob to be written verbatim, got %q", got.Spec.Config)
	}
}

// TestHandleVectorData_LazyRefetchPropagatesError covers the error path of the lazy re-fetch.
func TestHandleVectorData_LazyRefetchPropagatesError(t *testing.T) {
	vd := newTestVectorDeployment("vd-refetch-fail", nil)
	vd.Status.ResolvedVectorOcm = "non-empty-marker"

	stub := stubVectorOcmPort{
		getVectorDescriptor: func(_ context.Context, _ compref.Ref) (VectorDescriptor, error) {
			return VectorDescriptor{}, errors.New("registry unavailable")
		},
	}
	r, _ := newReconcilerForVectorDataTest(t, stub, vd)

	err := r.handleVectorData(context.Background(), vd, testVectorRef, nil, logf.Log)
	if err == nil {
		t.Fatalf("expected error from lazy re-fetch, got nil")
	}
	if !strings.Contains(err.Error(), "registry unavailable") {
		t.Errorf("expected wrapped adapter error, got: %v", err)
	}
	cond := meta.FindStatusCondition(vd.Status.Conditions, star.VectorDataCreatedCondition)
	if cond == nil || cond.Status != metav1.ConditionFalse || cond.Reason != "ConfigBlobRefetchFailed" {
		t.Errorf("expected VectorDataCreated=False/ConfigBlobRefetchFailed, got: %#v", cond)
	}
}

// TestVectorDataIsReady_ReportsImplementorState covers the readiness probe the Reconcile loop uses to gate
// VectorReady on the implementor having flipped VectorData.Status.Ready.
func TestVectorDataIsReady_ReportsImplementorState(t *testing.T) {
	vd := newTestVectorDeployment("vd-ready-probe", nil)
	vd.Status.ResultingVectorData = &star.LocalObjectReference{Name: "vd-ready-probe"}

	notReadyYet := &star.VectorData{
		ObjectMeta: metav1.ObjectMeta{Name: "vd-ready-probe", Namespace: "landscape-a"},
	}
	r, c := newReconcilerForVectorDataTest(t, nil, vd, notReadyYet)
	ctx := context.Background()

	ok, err := r.vectorDataIsReady(ctx, vd)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if ok {
		t.Errorf("expected ready=false before implementor flips the condition")
	}

	// Flip Ready=True and re-probe.
	meta.SetStatusCondition(&notReadyYet.Status.Conditions, metav1.Condition{
		Type:   star.VectorDataReadyCondition,
		Status: metav1.ConditionTrue,
		Reason: star.VectorDataReasonMaterialized,
	})
	if err := c.Status().Update(ctx, notReadyYet); err != nil {
		t.Fatalf("failed to update VectorData status: %v", err)
	}
	ok, err = r.vectorDataIsReady(ctx, vd)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !ok {
		t.Errorf("expected ready=true after implementor flips Ready")
	}
}

// TestHandleVectorDeploymentDeletion_RemovesVectorDataAndFinalizer covers the teardown contract: the
// vector-deployment-controller explicitly deletes the owned VectorData (which in turn cascades to whatever the
// runtime implementor produced) and then drops its own finalizer.
func TestHandleVectorDeploymentDeletion_RemovesVectorDataAndFinalizer(t *testing.T) {
	now := metav1.Now()
	vd := newTestVectorDeployment("vd-tearing-down", nil)
	vd.Finalizers = []string{star.VectorDataFinalizer}
	vd.DeletionTimestamp = &now

	vectorData := &star.VectorData{
		ObjectMeta: metav1.ObjectMeta{Name: "vd-tearing-down", Namespace: "landscape-a"},
	}
	r, c := newReconcilerForVectorDataTest(t, nil, vd, vectorData)
	ctx := context.Background()

	if _, err := r.handleVectorDeploymentDeletion(ctx, vd, logf.Log); err != nil {
		t.Fatalf("handleVectorDeploymentDeletion: %v", err)
	}

	got := &star.VectorData{}
	err := c.Get(ctx, types.NamespacedName{Name: "vd-tearing-down", Namespace: "landscape-a"}, got)
	if err == nil {
		t.Fatalf("expected VectorData to be deleted, but it is still present")
	}
	if !apierrors.IsNotFound(err) {
		t.Fatalf("unexpected error fetching deleted VectorData: %v", err)
	}
	for _, f := range vd.Finalizers {
		if f == star.VectorDataFinalizer {
			t.Errorf("expected VectorDataFinalizer to be removed, still present in %v", vd.Finalizers)
		}
	}
}

// TestHandleVectorDeploymentDeletion_NoOpWhenVectorDataAlreadyGone exercises the idempotent path: the runtime cascade
// may have already removed the VectorData. The handler must still drop the finalizer cleanly.
func TestHandleVectorDeploymentDeletion_NoOpWhenVectorDataAlreadyGone(t *testing.T) {
	now := metav1.Now()
	vd := newTestVectorDeployment("vd-already-gone", nil)
	vd.Finalizers = []string{star.VectorDataFinalizer}
	vd.DeletionTimestamp = &now

	r, _ := newReconcilerForVectorDataTest(t, nil, vd)
	ctx := context.Background()

	if _, err := r.handleVectorDeploymentDeletion(ctx, vd, logf.Log); err != nil {
		t.Fatalf("handleVectorDeploymentDeletion: %v", err)
	}
	for _, f := range vd.Finalizers {
		if f == star.VectorDataFinalizer {
			t.Errorf("expected VectorDataFinalizer to be removed even when VectorData was already gone")
		}
	}
}

// Compile-time touch so `sha256Hex` (used elsewhere by tests) doesn't get accidentally orphaned by future
// refactoring. The function is small but keeping it referenced means an import or signature change is caught.
var _ = sha256Hex

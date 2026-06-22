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

// nullJSON is the JSON literal written into ConfigMap keys whose source field is absent. Extracted to a constant so
// goconst stops yelling at us about the many appearances across these tests.
const nullJSON = "null"

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

// deploymentResultCMKey returns the ConfigMap data key under which the deployment result for the given component is
// expected to be written. Mirrors the production layout in handleVectorConfig / buildVectorConfigPayload: the last
// `/`-separated path segment is used as the suffix because the K8s ConfigMap key charset (`[-._a-zA-Z0-9]+`) rules
// out the full slashed component name.
func deploymentResultCMKey(component string) string {
	if idx := strings.LastIndex(component, "/"); idx >= 0 && idx != len(component)-1 {
		component = component[idx+1:]
	}
	return DeploymentResultsKeyPrefix + component + JSONSuffix
}

// TestHandleVectorConfig_EmptyState_WritesConfigMapWithNullKeys covers the zero-input path: no OCM envelope, no
// DeploymentResults. The ConfigMap must still be written so consumers can rely on its presence after VectorReady;
// `features.json` and `authored.json` carry the JSON literal `null`, and no `deploymentResults.*.json` keys exist.
func TestHandleVectorConfig_EmptyState_WritesConfigMapWithNullKeys(t *testing.T) {
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

	if got := cm.Data[FeaturesConfigKey]; got != nullJSON {
		t.Errorf("expected %s to be 'null', got %q", FeaturesConfigKey, got)
	}
	if got := cm.Data[AuthoredConfigKey]; got != nullJSON {
		t.Errorf("expected %s to be 'null', got %q", AuthoredConfigKey, got)
	}
	for key := range cm.Data {
		if strings.HasPrefix(key, DeploymentResultsKeyPrefix) {
			t.Errorf("expected no deploymentResults.* keys on empty state, got %q", key)
		}
	}
	if !meta.IsStatusConditionTrue(vd.Status.Conditions, star.VectorConfigCommittedCondition) {
		t.Errorf("expected VectorConfigCommittedCondition=True")
	}
	if vd.Status.ResolvedVectorConfigHash != "" {
		t.Errorf("expected empty ResolvedVectorConfigHash when there is no authored config, got %q",
			vd.Status.ResolvedVectorConfigHash)
	}
}

// TestHandleVectorConfig_EnvelopeSplitsIntoFeaturesAndAuthored asserts that the OCM envelope (the singleton
// `cloud-konfidence-vector-config` resource produced by the galaxy assembly side) is parsed into its `features` and
// `authored` fields, and that each is forwarded byte-for-byte into the matching ConfigMap key. The blob as a whole
// is hashed onto status for traceability.
func TestHandleVectorConfig_EnvelopeSplitsIntoFeaturesAndAuthored(t *testing.T) {
	features := `{"darkMode":true,"maxUsers":150}`
	authored := `{"database":{"host":"mysql","port":3306}}`
	envelope := []byte(`{"features":` + features + `,"authored":` + authored + `}`)

	vd := newTestVectorDeployment("vd-envelope", nil)
	r, c := newReconcilerForVectorConfigTest(t, nil, vd)

	if err := r.handleVectorConfig(context.Background(), vd, testVectorRef, envelope, logf.Log); err != nil {
		t.Fatalf("handleVectorConfig returned error: %v", err)
	}

	cm := &corev1.ConfigMap{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "vector-data-vd-envelope", Namespace: "landscape-a"}, cm); err != nil {
		t.Fatalf("expected ConfigMap to exist, got: %v", err)
	}
	if got := cm.Data[FeaturesConfigKey]; got != features {
		t.Errorf("expected features.json to be forwarded verbatim\n got:  %s\n want: %s", got, features)
	}
	if got := cm.Data[AuthoredConfigKey]; got != authored {
		t.Errorf("expected authored.json to be forwarded verbatim\n got:  %s\n want: %s", got, authored)
	}

	wantHash := sha256Hex(envelope)
	if vd.Status.ResolvedVectorConfigHash != wantHash {
		t.Errorf("expected ResolvedVectorConfigHash %q, got %q", wantHash, vd.Status.ResolvedVectorConfigHash)
	}
}

// TestHandleVectorConfig_EnvelopeWithOnlyFeatures asserts that omitting the `authored` field of the envelope yields
// `authored.json: null` while `features.json` carries the verbatim subset. Symmetric case to OnlyAuthored.
func TestHandleVectorConfig_EnvelopeWithOnlyFeatures(t *testing.T) {
	features := `{"darkMode":true}`
	envelope := []byte(`{"features":` + features + `}`)

	vd := newTestVectorDeployment("vd-only-features", nil)
	r, c := newReconcilerForVectorConfigTest(t, nil, vd)

	if err := r.handleVectorConfig(context.Background(), vd, testVectorRef, envelope, logf.Log); err != nil {
		t.Fatalf("handleVectorConfig returned error: %v", err)
	}

	cm := &corev1.ConfigMap{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "vector-data-vd-only-features", Namespace: "landscape-a"}, cm); err != nil {
		t.Fatalf("expected ConfigMap to exist: %v", err)
	}
	if got := cm.Data[FeaturesConfigKey]; got != features {
		t.Errorf("expected features.json %q, got %q", features, got)
	}
	if got := cm.Data[AuthoredConfigKey]; got != nullJSON {
		t.Errorf("expected authored.json to be 'null', got %q", got)
	}
}

// TestHandleVectorConfig_EnvelopeWithOnlyAuthored is the symmetric counterpart: only `authored` is set.
func TestHandleVectorConfig_EnvelopeWithOnlyAuthored(t *testing.T) {
	authored := `{"db":"postgres"}`
	envelope := []byte(`{"authored":` + authored + `}`)

	vd := newTestVectorDeployment("vd-only-authored", nil)
	r, c := newReconcilerForVectorConfigTest(t, nil, vd)

	if err := r.handleVectorConfig(context.Background(), vd, testVectorRef, envelope, logf.Log); err != nil {
		t.Fatalf("handleVectorConfig returned error: %v", err)
	}

	cm := &corev1.ConfigMap{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "vector-data-vd-only-authored", Namespace: "landscape-a"}, cm); err != nil {
		t.Fatalf("expected ConfigMap to exist: %v", err)
	}
	if got := cm.Data[FeaturesConfigKey]; got != nullJSON {
		t.Errorf("expected features.json to be 'null', got %q", got)
	}
	if got := cm.Data[AuthoredConfigKey]; got != authored {
		t.Errorf("expected authored.json %q, got %q", authored, got)
	}
}

// TestHandleVectorConfig_RejectsInvalidEnvelopeJSON exercises the validation gate: a malformed envelope must yield an
// error and flip VectorConfigCommitted to False rather than silently writing garbage into the landscape ConfigMap.
func TestHandleVectorConfig_RejectsInvalidEnvelopeJSON(t *testing.T) {
	vd := newTestVectorDeployment("vd-bad", nil)
	r, _ := newReconcilerForVectorConfigTest(t, nil, vd)

	err := r.handleVectorConfig(context.Background(), vd, testVectorRef, []byte("not json {"), logf.Log)
	if err == nil {
		t.Fatalf("expected error on invalid envelope JSON, got nil")
	}
	if cond := meta.FindStatusCondition(vd.Status.Conditions, star.VectorConfigCommittedCondition); cond == nil || cond.Status != metav1.ConditionFalse {
		t.Errorf("expected VectorConfigCommittedCondition=False on bad payload, got: %#v", cond)
	}
}

// TestHandleVectorConfig_DeploymentResultsOneKeyPerArtifact asserts the per-component layout expected by the central
// vector configuration service: each artifact contributes one `deploymentResults.<component>.json` key whose value
// is the deployer-emitted Spec JSON forwarded verbatim. Result names are NOT part of the key (the invariant on the
// Star side is one DeploymentResult per artifact).
func TestHandleVectorConfig_DeploymentResultsOneKeyPerArtifact(t *testing.T) {
	specA := `{"endpoint":"http://a"}`
	specB := `{"endpoint":"http://b"}`
	results := map[string]star.DeploymentResult{
		"svc-a/result-1": {Name: "result-1", Type: "test", Spec: runtime.RawExtension{Raw: []byte(specA)}},
		"svc-b/result-1": {Name: "result-1", Type: "test", Spec: runtime.RawExtension{Raw: []byte(specB)}},
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
	if got := cm.Data[deploymentResultCMKey("svc-a")]; got != specA {
		t.Errorf("expected %s to be %q, got %q", deploymentResultCMKey("svc-a"), specA, got)
	}
	if got := cm.Data[deploymentResultCMKey("svc-b")]; got != specB {
		t.Errorf("expected %s to be %q, got %q", deploymentResultCMKey("svc-b"), specB, got)
	}

	// Ensure the result name is not encoded into the key.
	for key := range cm.Data {
		if strings.HasPrefix(key, DeploymentResultsKeyPrefix) {
			suffix := strings.TrimSuffix(strings.TrimPrefix(key, DeploymentResultsKeyPrefix), JSONSuffix)
			if strings.Contains(suffix, "/") || strings.Contains(suffix, "__") {
				t.Errorf("expected component-only key, got composite-looking key %q", key)
			}
		}
	}
}

// TestHandleVectorConfig_RejectsMultipleResultsPerComponent guards the invariant that each artifact emits at most one
// DeploymentResult. If a deployer ever returns two results for the same component, the controller must error rather
// than silently overwrite, because the per-component CM layout cannot represent the second one.
func TestHandleVectorConfig_RejectsMultipleResultsPerComponent(t *testing.T) {
	results := map[string]star.DeploymentResult{
		"svc-a/result-1": {Name: "result-1", Spec: runtime.RawExtension{Raw: []byte(`{"a":1}`)}},
		"svc-a/result-2": {Name: "result-2", Spec: runtime.RawExtension{Raw: []byte(`{"a":2}`)}},
	}
	vd := newTestVectorDeployment("vd-dup", results)
	r, _ := newReconcilerForVectorConfigTest(t, nil, vd)

	err := r.handleVectorConfig(context.Background(), vd, testVectorRef, nil, logf.Log)
	if err == nil {
		t.Fatalf("expected error when an artifact emits more than one DeploymentResult, got nil")
	}
	if !strings.Contains(err.Error(), "more than one DeploymentResult") {
		t.Errorf("expected wrapped invariant error, got: %v", err)
	}
	if cond := meta.FindStatusCondition(vd.Status.Conditions, star.VectorConfigCommittedCondition); cond == nil || cond.Status != metav1.ConditionFalse {
		t.Errorf("expected VectorConfigCommittedCondition=False on invariant violation, got: %#v", cond)
	}
}

// TestHandleVectorConfig_SlashedComponentNamesUseBasename mirrors the real-world setup where OCM component names
// contain slashes (e.g. `github.com/konfidence-project/sample-service-1`). Slashes are not legal in ConfigMap data
// keys, so the controller writes under the last path segment. This test locks the key layout in.
func TestHandleVectorConfig_SlashedComponentNamesUseBasename(t *testing.T) {
	const fullComponent = "github.com/konfidence-project/sample-service-1"
	const basename = "sample-service-1"
	spec := `{"endpoint":"http://x"}`
	results := map[string]star.DeploymentResult{
		fullComponent + "/main": {Name: "main", Spec: runtime.RawExtension{Raw: []byte(spec)}},
	}
	vd := newTestVectorDeployment("vd-slashed", results)
	r, c := newReconcilerForVectorConfigTest(t, nil, vd)

	if err := r.handleVectorConfig(context.Background(), vd, testVectorRef, nil, logf.Log); err != nil {
		t.Fatalf("handleVectorConfig returned error: %v", err)
	}

	cm := &corev1.ConfigMap{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "vector-data-vd-slashed", Namespace: "landscape-a"}, cm); err != nil {
		t.Fatalf("expected ConfigMap to exist: %v", err)
	}
	want := DeploymentResultsKeyPrefix + basename + JSONSuffix
	if got := cm.Data[want]; got != spec {
		t.Errorf("expected key %q to be %q, got %q (full data: %#v)", want, spec, got, cm.Data)
	}
	// Ensure the full slashed component name was NOT used as a key (would be an invalid CM anyway, but
	// double-checking guards against accidental regression).
	for key := range cm.Data {
		if strings.Contains(key, "/") {
			t.Errorf("ConfigMap key %q contains an illegal '/' character", key)
		}
	}
}

// TestHandleVectorConfig_RejectsComponentBasenameCollision catches the rare case where two distinct OCM components
// share the same last path segment (e.g. `foo/bar/svc` and `foo/baz/svc` both collapse to `svc`). Silently dropping
// one of the results would violate the "one CM key per artifact" guarantee, so this is treated as an authoring error.
func TestHandleVectorConfig_RejectsComponentBasenameCollision(t *testing.T) {
	results := map[string]star.DeploymentResult{
		"foo/bar/svc/main": {Name: "main", Spec: runtime.RawExtension{Raw: []byte(`{"a":1}`)}},
		"foo/baz/svc/main": {Name: "main", Spec: runtime.RawExtension{Raw: []byte(`{"a":2}`)}},
	}
	vd := newTestVectorDeployment("vd-basename-collision", results)
	r, _ := newReconcilerForVectorConfigTest(t, nil, vd)

	err := r.handleVectorConfig(context.Background(), vd, testVectorRef, nil, logf.Log)
	if err == nil {
		t.Fatalf("expected basename-collision error, got nil")
	}
	if !strings.Contains(err.Error(), "same ConfigMap-key basename") {
		t.Errorf("expected collision diagnostic, got: %v", err)
	}
}

// TestHandleVectorConfig_EmptyDeploymentResultSpecBecomesNull covers the corner case where a deployer marks a result
// without populating Spec. The CM value must still be valid JSON; we write `null` rather than an empty string.
func TestHandleVectorConfig_EmptyDeploymentResultSpecBecomesNull(t *testing.T) {
	results := map[string]star.DeploymentResult{
		"svc-a/result-1": {Name: "result-1", Type: "test", Spec: runtime.RawExtension{}},
	}
	vd := newTestVectorDeployment("vd-empty-spec", results)
	r, c := newReconcilerForVectorConfigTest(t, nil, vd)

	if err := r.handleVectorConfig(context.Background(), vd, testVectorRef, nil, logf.Log); err != nil {
		t.Fatalf("handleVectorConfig returned error: %v", err)
	}

	cm := &corev1.ConfigMap{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "vector-data-vd-empty-spec", Namespace: "landscape-a"}, cm); err != nil {
		t.Fatalf("expected ConfigMap to exist: %v", err)
	}
	if got := cm.Data[deploymentResultCMKey("svc-a")]; got != nullJSON {
		t.Errorf("expected null spec, got %q", got)
	}
}

// TestHandleVectorConfig_NoOpWhenAlreadyPresent asserts the function trusts an existing ConfigMap and does not attempt
// to mutate it. Vector data is immutable per ADR-0024 -- once written, the controller has no business touching it.
func TestHandleVectorConfig_NoOpWhenAlreadyPresent(t *testing.T) {
	vd := newTestVectorDeployment("vd-existing", nil)

	preExistingFeatures := `{"prior":true}`
	preExisting := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "vector-data-vd-existing",
			Namespace: "landscape-a",
			Labels: map[string]string{
				pkgctrl.ManagedByLabel: VectorDeploymentControllerName,
			},
		},
		Data: map[string]string{
			FeaturesConfigKey: preExistingFeatures,
			AuthoredConfigKey: nullJSON,
		},
	}

	r, c := newReconcilerForVectorConfigTest(t, nil, vd, preExisting)

	if err := r.handleVectorConfig(context.Background(), vd, testVectorRef, []byte(`{"features":{"would-overwrite":true}}`), logf.Log); err != nil {
		t.Fatalf("handleVectorConfig returned error: %v", err)
	}

	got := &corev1.ConfigMap{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "vector-data-vd-existing", Namespace: "landscape-a"}, got); err != nil {
		t.Fatalf("expected ConfigMap to still exist: %v", err)
	}
	if got.Data[FeaturesConfigKey] != preExistingFeatures {
		t.Errorf("expected pre-existing features.json to be preserved, got %q", got.Data[FeaturesConfigKey])
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
	features := `{"a":1}`
	expectedBlob := []byte(`{"features":` + features + `}`)

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
	if got := cm.Data[FeaturesConfigKey]; got != features {
		t.Errorf("expected re-fetched features to be written verbatim, got %q", got)
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

// TestSplitConfigEnvelope is a focused unit test on the envelope-splitting helper, covering the cases that
// handleVectorConfig depends on but is awkward to assert at the controller level.
func TestSplitConfigEnvelope(t *testing.T) {
	cases := []struct {
		name              string
		input             []byte
		wantFeatures      string
		wantAuthored      string
		wantErrSubstring  string
		wantValidFeatures bool
		wantValidAuthored bool
	}{
		{name: "nil input is both null", input: nil, wantFeatures: nullJSON, wantAuthored: nullJSON},
		{name: "empty input is both null", input: []byte{}, wantFeatures: nullJSON, wantAuthored: nullJSON},
		{name: "both fields present", input: []byte(`{"features":{"x":1},"authored":{"y":2}}`),
			wantFeatures: `{"x":1}`, wantAuthored: `{"y":2}`},
		{name: "only features", input: []byte(`{"features":{"x":1}}`), wantFeatures: `{"x":1}`, wantAuthored: nullJSON},
		{name: "only authored", input: []byte(`{"authored":{"y":2}}`), wantFeatures: nullJSON, wantAuthored: `{"y":2}`},
		{name: "empty object", input: []byte(`{}`), wantFeatures: nullJSON, wantAuthored: nullJSON},
		{name: "invalid json rejected", input: []byte(`not json`), wantErrSubstring: "not valid JSON"},
		{name: "valid json but wrong shape rejected", input: []byte(`["features","authored"]`), wantErrSubstring: "envelope shape"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			features, authored, err := splitConfigEnvelope(tc.input)
			if tc.wantErrSubstring != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErrSubstring)
				}
				if !strings.Contains(err.Error(), tc.wantErrSubstring) {
					t.Errorf("error %q does not contain %q", err.Error(), tc.wantErrSubstring)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if features != tc.wantFeatures {
				t.Errorf("features mismatch:\n got:  %s\n want: %s", features, tc.wantFeatures)
			}
			if authored != tc.wantAuthored {
				t.Errorf("authored mismatch:\n got:  %s\n want: %s", authored, tc.wantAuthored)
			}
			// Whatever we return must always be valid JSON; verify.
			var anyVal any
			if err := json.Unmarshal([]byte(features), &anyVal); err != nil {
				t.Errorf("features is not valid JSON: %v", err)
			}
			if err := json.Unmarshal([]byte(authored), &anyVal); err != nil {
				t.Errorf("authored is not valid JSON: %v", err)
			}
		})
	}
}

// TestSplitDeploymentResultKey locks in the shape that handleArtifactDeployments produces. Any future change that
// alters the composite layout must also update buildVectorConfigPayload and this test in lockstep.
func TestSplitDeploymentResultKey(t *testing.T) {
	cases := []struct {
		input                string
		wantComponent        string
		wantResultName       string
		wantErr              bool
		wantErrSubstring     string
		wantErrSubstringElse string
	}{
		{input: "svc-a/main", wantComponent: "svc-a", wantResultName: "main"},
		{input: "github.com/foo/bar/baz/result", wantComponent: "github.com/foo/bar/baz", wantResultName: "result"},
		{input: "no-slash", wantErr: true, wantErrSubstring: "<component>/<resultName>"},
		{input: "/missing-component", wantErr: true, wantErrSubstring: "empty component"},
		{input: "missing-result/", wantErr: true, wantErrSubstring: "empty component"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			comp, name, err := splitDeploymentResultKey(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if tc.wantErrSubstring != "" && !strings.Contains(err.Error(), tc.wantErrSubstring) {
					t.Errorf("error %q does not contain %q", err.Error(), tc.wantErrSubstring)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if comp != tc.wantComponent {
				t.Errorf("component %q, want %q", comp, tc.wantComponent)
			}
			if name != tc.wantResultName {
				t.Errorf("resultName %q, want %q", name, tc.wantResultName)
			}
		})
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
			FeaturesConfigKey: nullJSON,
			AuthoredConfigKey: nullJSON,
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

package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	star "github.com/konfidence-project/konfidence/api/star/v1alpha1"
	pkgctrl "github.com/konfidence-project/konfidence/pkg/controller"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	// VectorDataConfigMapNamePrefix is the fixed prefix used for the per-vector data ConfigMap. The full name is
	// "vector-data-<vectorDeploymentName>", written into the landscape namespace (the namespace of the
	// VectorDeployment itself, per ADR-0007).
	//
	// NB: the merged per-landscape vector configuration service (kubernetes-landscape-orchestrator PR #12) currently
	// expects the ConfigMap name to be the bare `vectorId` (the OFREP `targetingKey`). The producer side intentionally
	// keeps the `vector-data-` prefix for now — the service will be aligned in a follow-up.
	VectorDataConfigMapNamePrefix = "vector-data-"

	// FeaturesConfigKey carries the OFREP feature flag map extracted from the `features` field of the OCM envelope
	// (the singleton resource named "cloud-konfidence-vector-config" produced by the galaxy assembly side). Always
	// present; written as the JSON literal `null` when the envelope did not declare features.
	FeaturesConfigKey = "features.json"

	// AuthoredConfigKey carries the authored configuration map extracted from the `authored` field of the OCM
	// envelope. Always present; written as the JSON literal `null` when the envelope did not declare authored data.
	AuthoredConfigKey = "authored.json"

	// DeploymentResultsKeyPrefix is the prefix under which per-artifact deployment results are written. Each artifact
	// contributes one key of the form "deploymentResults.<component>.json", carrying the deployer-emitted spec JSON
	// verbatim. No key is written when an artifact produced no results. The layout mirrors the merged per-landscape
	// vector configuration service (kubernetes-landscape-orchestrator PR #12) so that the service can address each
	// deployment independently.
	DeploymentResultsKeyPrefix = "deploymentResults."

	// JSONSuffix is the trailing suffix used for every JSON-valued ConfigMap key in this package.
	JSONSuffix = ".json"

	// jsonNull is the JSON literal written into ConfigMap keys whose source field is absent or empty (e.g. a vector
	// that declared no `features` block in its OCM envelope still gets a `features.json: null` entry). Always-present
	// keys keep the consumer side simple.
	jsonNull = "null"
)

// configEnvelope is the on-wire shape of the `cloud-konfidence-vector-config` OCM resource produced by the galaxy
// assembly side (see api/galaxy/v1alpha1/vector_config_types.go). Both fields are optional; the galaxy side omits
// the resource entirely when neither is set, but defensively we still accept either being nil on the consume side.
type configEnvelope struct {
	Features json.RawMessage `json:"features,omitempty"`
	Authored json.RawMessage `json:"authored,omitempty"`
}

// handleVectorConfig materializes the vector-scoped configuration ConfigMap in the landscape namespace.
//
// It is invoked after all ArtifactDeployments have reached Ready (so DeploymentResults are observable) and is a
// singleton write per VectorDeployment. The ConfigMap content is derived exclusively from data that is immutable for
// the lifetime of the VectorDeployment:
//
//   - VectorDeployment.Spec is immutable (XValidation rule), so the referenced vector cannot change.
//   - The vector OCM ComponentVersion is immutable, so the optional authored configuration envelope is fixed.
//   - ArtifactDeployment.Spec is immutable and DeploymentResults are documented as immutable per generation, so the
//     aggregated results gathered by handleArtifactDeployments are stable from the moment all artifacts go Ready.
//
// Consequently the function never updates an existing ConfigMap. If one is already present we trust it and no-op; if
// it is missing (first reconcile after VectorReady, or it was deleted out-of-band) we create it as Immutable=true with
// a controller-owner reference back to the VectorDeployment so that cleanup cascades on VD deletion.
//
// The “freshConfig“ parameter carries the authored configuration envelope if it was just fetched from OCM in this
// reconcile pass; on subsequent reconciles the caller passes nil and we re-fetch via the OCM adapter only when needed
// (i.e. when the ConfigMap is missing). This keeps the (potentially large) envelope out of the VD “Status“ — only
// its hash is persisted there for traceability.
func (r *VectorDeploymentReconciler) handleVectorConfig(
	ctx context.Context,
	vd *star.VectorDeployment,
	vectorRef compref.Ref,
	freshConfig []byte,
	log logr.Logger,
) error {
	cmName := vectorConfigConfigMapName(vd.Name)
	cmKey := types.NamespacedName{Namespace: vd.Namespace, Name: cmName}

	existing := &corev1.ConfigMap{}
	getErr := r.Get(ctx, cmKey, existing)
	switch {
	case getErr == nil:
		// Already written and immutable; trust it and move on.
		r.setVectorConfigCondition(vd, metav1.ConditionTrue, "Committed",
			fmt.Sprintf("Vector data ConfigMap %s already present", cmName))
		return nil
	case !apierrors.IsNotFound(getErr):
		r.setVectorConfigCondition(vd, metav1.ConditionFalse, "ConfigMapGetFailed", getErr.Error())
		return fmt.Errorf("failed to get vector data ConfigMap %s: %w", cmKey, getErr)
	}

	// Lazy re-fetch path: if we don't already have the envelope in scope (later reconciles where Reconcile took the
	// cached `Status.ResolvedVectorOcm` shortcut), pull the descriptor again. Cheap on the happy path because we
	// only enter this branch when the ConfigMap is genuinely missing.
	configBlob := freshConfig
	if configBlob == nil && vd.Status.ResolvedVectorConfigHash != "" {
		descr, err := r.OcmAdapter.GetVectorDescriptor(ctx, vectorRef)
		if err != nil {
			r.setVectorConfigCondition(vd, metav1.ConditionFalse, "ConfigBlobRefetchFailed", err.Error())
			return fmt.Errorf("failed to re-fetch vector config blob from OCM for %s: %w", cmKey, err)
		}
		configBlob = descr.Configuration
	}

	data, err := buildVectorConfigPayload(configBlob, vd.Status.DeploymentResults)
	if err != nil {
		r.setVectorConfigCondition(vd, metav1.ConditionFalse, "PayloadBuildFailed", err.Error())
		return err
	}

	immutable := true
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: vd.Namespace,
			Labels: map[string]string{
				pkgctrl.ManagedByLabel:            VectorDeploymentControllerName,
				pkgctrl.VectorDeploymentNameLabel: vd.Name,
				pkgctrl.VectorDeploymentUIDLabel:  string(vd.UID),
			},
		},
		Immutable: &immutable,
		Data:      data,
	}

	if err := controllerutil.SetControllerReference(vd, cm, r.Scheme); err != nil {
		r.setVectorConfigCondition(vd, metav1.ConditionFalse, "OwnerRefFailed", err.Error())
		return fmt.Errorf("failed to set controller reference on vector data ConfigMap %s: %w", cmKey, err)
	}

	if err := r.Create(ctx, cm); err != nil {
		r.setVectorConfigCondition(vd, metav1.ConditionFalse, "ConfigMapCreateFailed", err.Error())
		return fmt.Errorf("failed to create vector data ConfigMap %s: %w", cmKey, err)
	}

	// Persist the small fingerprint so operators can correlate `ResolvedVectorConfigHash` on the VD with the
	// `konfidence.cloud/vector-config-hash`-style annotation that future tooling may add. We never compare against
	// this value for drift detection (the vector is immutable, so by construction the hash on a successfully
	// committed CM is stable).
	if configBlob != nil {
		vd.Status.ResolvedVectorConfigHash = sha256Hex(configBlob)
	}

	r.setVectorConfigCondition(vd, metav1.ConditionTrue, "Committed",
		fmt.Sprintf("Vector data ConfigMap %s materialized", cmName))
	r.Recorder.Eventf(vd, nil, corev1.EventTypeNormal,
		"VectorConfigCommitted", "VectorConfigCommitted",
		fmt.Sprintf("Materialized vector data ConfigMap %s in namespace %s", cmName, vd.Namespace))
	log.Info("vector data ConfigMap committed", "configMap", cmKey.String())
	return nil
}

// buildVectorConfigPayload assembles the ConfigMap `Data` map from the in-memory inputs.
//
// The OCM envelope is unmarshalled into its two top-level fields and each is written to its own key (`features.json`,
// `authored.json`). Both keys are always present; absent fields become the JSON literal `null`. Each artifact's
// DeploymentResult is written to `deploymentResults.<component-basename>.json` carrying the deployer-emitted spec
// JSON verbatim. No `deploymentResults.*.json` key is written when an artifact produced no results.
//
// Component identifier: the OCM component name (e.g. `github.com/konfidence-project/sample-service-1`) cannot be used
// as a ConfigMap data key because the K8s key charset is `[-._a-zA-Z0-9]+` and most component names contain slashes.
// We use the last path segment instead (`sample-service-1`), mirroring how `ConstructArtifactDeploymentName` already
// derives the per-artifact identifier (see util.go). The merged per-landscape vector configuration service
// (kubernetes-landscape-orchestrator PR #12) treats the suffix as an opaque "deployment name" so any K8s-safe
// identifier works as long as it stays unique within the vector.
//
// Aggregation invariant: each artifact (i.e. each `component`) emits at most one `DeploymentResult`. The aggregated
// `vd.Status.DeploymentResults` is therefore keyed `"<component>/<resultName>"` with at most one entry per
// `<component>` (see internal/star/vectordeployment/internal/controller/vectordeployment_controller.go:289). We
// additionally guard against two distinct components collapsing to the same basename — that would silently overwrite
// one result with another and is treated as an error.
func buildVectorConfigPayload(
	configBlob []byte,
	deploymentResults map[string]star.DeploymentResult,
) (map[string]string, error) {
	data := make(map[string]string, 2+len(deploymentResults))

	// Envelope split.
	featuresValue, authoredValue, err := splitConfigEnvelope(configBlob)
	if err != nil {
		return nil, err
	}
	data[FeaturesConfigKey] = featuresValue
	data[AuthoredConfigKey] = authoredValue

	// Deployment results: one CM key per component, value is the result's Spec JSON verbatim.
	//
	// `seenComponents` maps the K8s-safe basename back to the full component name that produced it, so collisions
	// produce an informative error rather than silent data loss.
	seenComponents := make(map[string]string, len(deploymentResults))
	for compositeKey, result := range deploymentResults {
		component, _, err := splitDeploymentResultKey(compositeKey)
		if err != nil {
			return nil, err
		}
		basename := componentBasename(component)
		if prev, exists := seenComponents[basename]; exists {
			if prev == component {
				// Two DeploymentResults under the same component is an invariant violation (one-result-per-artifact)
				// that the per-result loop guards against; we should never get here, but the diagnostic is clearer.
				return nil, fmt.Errorf(
					"component %q emitted more than one DeploymentResult; expected at most one per artifact", component,
				)
			}
			return nil, fmt.Errorf(
				"components %q and %q share the same ConfigMap-key basename %q; rename one of them or address them by full name",
				prev, component, basename,
			)
		}
		seenComponents[basename] = component

		cmKey := DeploymentResultsKeyPrefix + basename + JSONSuffix
		value, err := deploymentResultValue(result)
		if err != nil {
			return nil, fmt.Errorf("invalid deployment result spec for component %q: %w", component, err)
		}
		data[cmKey] = value
	}

	return data, nil
}

// componentBasename returns the last `/`-separated segment of an OCM component name so the result is K8s-safe for use
// as a ConfigMap data key. Mirrors the basename heuristic in ConstructArtifactDeploymentName (util.go:49-54).
func componentBasename(component string) string {
	idx := strings.LastIndex(component, "/")
	if idx < 0 || idx == len(component)-1 {
		return component
	}
	return component[idx+1:]
}

// splitConfigEnvelope unmarshals the OCM envelope and returns the verbatim JSON for `features` and `authored`.
// Either may be the literal "null" when the corresponding field is absent or itself JSON null. When the input is
// empty (the vector did not declare a config resource at all) both return values are "null".
func splitConfigEnvelope(configBlob []byte) (features, authored string, err error) {
	if len(configBlob) == 0 {
		return jsonNull, jsonNull, nil
	}
	if !json.Valid(configBlob) {
		return "", "", fmt.Errorf("vector configuration payload is not valid JSON")
	}
	var env configEnvelope
	if err := json.Unmarshal(configBlob, &env); err != nil {
		return "", "", fmt.Errorf("vector configuration payload is not a JSON object with the expected envelope shape: %w", err)
	}
	features = rawOrNull(env.Features)
	authored = rawOrNull(env.Authored)
	return features, authored, nil
}

// rawOrNull renders a json.RawMessage as its verbatim text, or "null" when the field was absent.
func rawOrNull(r json.RawMessage) string {
	if len(r) == 0 {
		return jsonNull
	}
	return string(r)
}

// splitDeploymentResultKey splits the aggregated key produced in handleArtifactDeployments
// (`<component>/<resultName>`). The result name is currently discarded for the ConfigMap layout because the central
// vector configuration service addresses by component, but we still return it so the caller can include it in any
// future error message or diagnostic.
func splitDeploymentResultKey(compositeKey string) (component, resultName string, err error) {
	idx := strings.LastIndex(compositeKey, "/")
	if idx < 0 {
		return "", "", fmt.Errorf("malformed deployment result key %q: expected <component>/<resultName>", compositeKey)
	}
	component = compositeKey[:idx]
	resultName = compositeKey[idx+1:]
	if component == "" || resultName == "" {
		return "", "", fmt.Errorf("malformed deployment result key %q: empty component or result name", compositeKey)
	}
	return component, resultName, nil
}

// deploymentResultValue renders the deployer-emitted spec JSON for the ConfigMap value. When the deployer left Spec
// empty, "null" is written so the CM value is still valid JSON.
func deploymentResultValue(result star.DeploymentResult) (string, error) {
	if len(result.Spec.Raw) == 0 {
		return jsonNull, nil
	}
	if !json.Valid(result.Spec.Raw) {
		return "", fmt.Errorf("spec is not valid JSON")
	}
	return string(result.Spec.Raw), nil
}

// vectorConfigConfigMapName returns the deterministic ConfigMap name for a given VectorDeployment. Names are unique
// per landscape namespace, mirroring how VectorAssignment is keyed by the VectorDeployment name.
func vectorConfigConfigMapName(vdName string) string {
	return VectorDataConfigMapNamePrefix + vdName
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// setVectorConfigCondition is a small helper to keep the condition shape consistent across the success and failure
// paths in handleVectorConfig.
func (r *VectorDeploymentReconciler) setVectorConfigCondition(
	vd *star.VectorDeployment,
	status metav1.ConditionStatus,
	reason, message string,
) {
	meta.SetStatusCondition(&vd.Status.Conditions, metav1.Condition{
		Type:               star.VectorConfigCommittedCondition,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: vd.Generation,
		LastTransitionTime: metav1.Now(),
	})
}

// handleVectorDeploymentDeletion drives the teardown sequence when the VectorDeployment is being deleted. It deletes
// the vector data ConfigMap explicitly (in addition to the controller-owner-reference cascade that Kubernetes performs
// in the background) and then removes the VectorDataFinalizer so that the API server can finalize the VD object. The
// function is idempotent: if the ConfigMap is already gone, the delete is a no-op; if the finalizer is already absent,
// no patch is issued.
func (r *VectorDeploymentReconciler) handleVectorDeploymentDeletion(
	ctx context.Context,
	vd *star.VectorDeployment,
	log logr.Logger,
) (ctrl.Result, error) {
	if !controllerutil.ContainsFinalizer(vd, star.VectorDataFinalizer) {
		// Nothing left for us to do; the API server will finish removing the object.
		return ctrl.Result{}, nil
	}

	cmName := vectorConfigConfigMapName(vd.Name)
	cmKey := types.NamespacedName{Namespace: vd.Namespace, Name: cmName}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: cmName, Namespace: vd.Namespace},
	}
	if err := r.Delete(ctx, cm); err != nil && !apierrors.IsNotFound(err) {
		return ctrl.Result{}, fmt.Errorf("failed to delete vector data ConfigMap %s during teardown: %w", cmKey, err)
	}
	log.Info("vector data ConfigMap removed during VectorDeployment teardown", "configMap", cmKey.String())

	patch := client.MergeFrom(vd.DeepCopy())
	controllerutil.RemoveFinalizer(vd, star.VectorDataFinalizer)
	if err := r.Patch(ctx, vd, patch); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to remove vector-data finalizer from %s/%s: %w", vd.Namespace, vd.Name, err)
	}
	return ctrl.Result{}, nil
}

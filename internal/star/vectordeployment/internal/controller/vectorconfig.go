package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

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
	VectorDataConfigMapNamePrefix = "vector-data-"

	// VectorConfigDataKey carries the optional authored configuration blob (the bytes of the OCM resource named
	// "cloud-konfidence-vector-config") verbatim. It is written as `null` when the vector does not declare such
	// a resource. Two distinct keys (instead of one bundled `data.json`) keep the two semantically-independent
	// concerns separately accessible — both for `kubectl get cm -o jsonpath` and for the future per-landscape
	// vector data service which can serve them on independent endpoints.
	VectorConfigDataKey = "config.json"

	// VectorDeploymentResultsDataKey carries the aggregated DeploymentResults of all underlying ArtifactDeployments,
	// keyed "<componentName>/<resultName>". Always present, materialized as `{}` when no artifact has produced any
	// results, so consumers do not have to special-case the zero state.
	VectorDeploymentResultsDataKey = "deployment-results.json"
)

// handleVectorConfig materializes the vector-scoped configuration ConfigMap in the landscape namespace.
//
// It is invoked after all ArtifactDeployments have reached Ready (so DeploymentResults are observable) and is a
// singleton write per VectorDeployment. The ConfigMap content is derived exclusively from data that is immutable for
// the lifetime of the VectorDeployment:
//
//   - VectorDeployment.Spec is immutable (XValidation rule), so the referenced vector cannot change.
//   - The vector OCM ComponentVersion is immutable, so the optional authored configuration blob is fixed.
//   - ArtifactDeployment.Spec is immutable and DeploymentResults are documented as immutable per generation, so the
//     aggregated results gathered by handleArtifactDeployments are stable from the moment all artifacts go Ready.
//
// Consequently the function never updates an existing ConfigMap. If one is already present we trust it and no-op; if
// it is missing (first reconcile after VectorReady, or it was deleted out-of-band) we create it as Immutable=true with
// a controller-owner reference back to the VectorDeployment so that cleanup cascades on VD deletion.
//
// The “freshConfig“ parameter carries the authored configuration blob if it was just fetched from OCM in this
// reconcile pass; on subsequent reconciles the caller passes nil and we re-fetch via the OCM adapter only when needed
// (i.e. when the ConfigMap is missing). This keeps the (potentially large) authored blob out of the VD “Status“ —
// only its hash is persisted there for traceability.
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

	// Lazy re-fetch path: if we don't already have the blob in scope (later reconciles where Reconcile took the
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

	configValue, resultsValue, err := buildVectorConfigPayload(configBlob, vd.Status.DeploymentResults)
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
		Data: map[string]string{
			VectorConfigDataKey:            configValue,
			VectorDeploymentResultsDataKey: resultsValue,
		},
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

// buildVectorConfigPayload renders the two ConfigMap data values from the in-memory inputs.
//
// The authored configuration blob is forwarded unchanged when present and treated as the JSON literal `null`
// otherwise. DeploymentResults are materialized as `{}` (rather than null) when no artifact has produced any.
// Both return values are valid JSON documents on every code path.
func buildVectorConfigPayload(
	configBlob []byte,
	deploymentResults map[string]star.DeploymentResult,
) (string, string, error) {
	var configValue string
	if len(configBlob) == 0 {
		configValue = "null"
	} else {
		if !json.Valid(configBlob) {
			return "", "", fmt.Errorf("vector configuration payload is not valid JSON")
		}
		configValue = string(configBlob)
	}

	if deploymentResults == nil {
		deploymentResults = map[string]star.DeploymentResult{}
	}
	resultsBytes, err := json.Marshal(deploymentResults)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal deployment results: %w", err)
	}

	return configValue, string(resultsBytes), nil
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

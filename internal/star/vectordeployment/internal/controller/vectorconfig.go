package controller

import (
	"context"
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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	// VectorDataConfigMapNamePrefix is the fixed prefix used for the per-vector data ConfigMap. The full name is
	// "vector-data-<vectorDeploymentName>", written into the landscape namespace (the namespace of the
	// VectorDeployment itself, per ADR-0007).
	VectorDataConfigMapNamePrefix = "vector-data-"

	// VectorConfigDataKey is the single key under which the canonical JSON payload is stored in the ConfigMap.
	VectorConfigDataKey = "data.json"
)

// vectorConfigPayload is the on-wire shape persisted into the ConfigMap. The payload is computed once per
// VectorDeployment and is immutable for the lifetime of that deployment (the vector OCM ComponentVersion is immutable
// and ArtifactDeployment.spec is also immutable -- both inputs that feed this struct are therefore fixed by the time
// handleVectorConfig is invoked). Field ordering is fixed for stable JSON marshalling.
type vectorConfigPayload struct {
	// Config carries the optional authored vector-scoped configuration blob (the bytes of the OCM resource named
	// "cloud-konfidence-vector-config") verbatim. It is null when the vector did not declare such a resource.
	Config json.RawMessage `json:"config"`
	// DeploymentResults is the aggregated set of results produced by all underlying ArtifactDeployments, keyed
	// "<componentName>/<resultName>". An empty map is materialized as "{}" rather than null so that consumers do not
	// have to special-case the zero state.
	DeploymentResults map[string]star.DeploymentResult `json:"deploymentResults"`
}

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
// Consequently the function never needs to update an existing ConfigMap. If one is already present we trust it and
// no-op; if it is missing (first reconcile after VectorReady, or it was deleted out-of-band) we create it as
// Immutable=true with a controller-owner reference back to the VectorDeployment so that cleanup cascades on VD
// deletion.
func (r *VectorDeploymentReconciler) handleVectorConfig(
	ctx context.Context,
	vd *star.VectorDeployment,
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

	payload, err := buildVectorConfigPayload(vd)
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
			VectorConfigDataKey: string(payload),
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

	r.setVectorConfigCondition(vd, metav1.ConditionTrue, "Committed",
		fmt.Sprintf("Vector data ConfigMap %s materialized", cmName))
	r.Recorder.Eventf(vd, nil, corev1.EventTypeNormal,
		"VectorConfigCommitted", "VectorConfigCommitted",
		fmt.Sprintf("Materialized vector data ConfigMap %s in namespace %s", cmName, vd.Namespace))
	log.Info("vector data ConfigMap committed", "configMap", cmKey.String())
	return nil
}

// buildVectorConfigPayload assembles the canonical JSON written to the vector data ConfigMap. The authored configuration
// blob is forwarded unchanged when present and treated as null otherwise. DeploymentResults are materialized as an empty
// map (rather than null) when no artifact has produced any.
func buildVectorConfigPayload(vd *star.VectorDeployment) ([]byte, error) {
	payload := vectorConfigPayload{
		DeploymentResults: vd.Status.DeploymentResults,
	}
	if payload.DeploymentResults == nil {
		payload.DeploymentResults = map[string]star.DeploymentResult{}
	}
	if vd.Status.ResolvedVectorConfig != "" {
		raw := json.RawMessage(vd.Status.ResolvedVectorConfig)
		// Validate that the authored payload is itself valid JSON; otherwise the resulting ConfigMap content would be
		// undecodable for downstream consumers.
		if !json.Valid(raw) {
			return nil, fmt.Errorf("vector configuration payload is not valid JSON")
		}
		payload.Config = raw
	} else {
		payload.Config = json.RawMessage("null")
	}
	return json.Marshal(payload)
}

// vectorConfigConfigMapName returns the deterministic ConfigMap name for a given VectorDeployment. Names are unique
// per landscape namespace, mirroring how VectorAssignment is keyed by the VectorDeployment name.
func vectorConfigConfigMapName(vdName string) string {
	return VectorDataConfigMapNamePrefix + vdName
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

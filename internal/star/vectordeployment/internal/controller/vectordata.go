package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
	star "github.com/konfidence-project/konfidence/api/star/v1alpha1"
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

// handleVectorData materialises the runtime-agnostic VectorData CR that is the contract between the Star side
// (which resolves the OCM payload) and the runtime-specific implementor (which writes the data into the target
// runtime — e.g. as a Kubernetes ConfigMap).
//
// It is invoked after all ArtifactDeployments of the vector are Ready (so DeploymentResults are observable) and is
// a singleton write per VectorDeployment. The CR content is derived exclusively from data that is immutable for the
// lifetime of the VectorDeployment:
//
//   - VectorDeployment.Spec is immutable (XValidation rule), so the referenced vector cannot change.
//   - The vector OCM ComponentVersion is immutable, so the optional authored configuration blob is fixed.
//   - ArtifactDeployment.Spec is immutable and DeploymentResults are documented as immutable per generation, so the
//     aggregated results gathered by handleArtifactDeployments are stable from the moment all artifacts go Ready.
//
// Consequently the function never updates an existing VectorData. If one already exists we trust it and no-op; if it
// is missing (first reconcile after VectorDeployed, or it was deleted out-of-band) we create it with a
// controller-owner reference back to the VectorDeployment.
//
// The “freshConfig“ parameter carries the authored configuration blob if it was just fetched from OCM in this
// reconcile pass; on subsequent reconciles the caller passes nil and we re-fetch via the OCM adapter only when needed
// (i.e. when the VectorData is missing). This keeps the (potentially large) authored blob out of the VectorDeployment
// “Status“; the canonical home for the bytes is the VectorData CR itself.
func (r *VectorDeploymentReconciler) handleVectorData(
	ctx context.Context,
	vd *star.VectorDeployment,
	vectorRef compref.Ref,
	freshConfig []byte,
	log logr.Logger,
) error {
	vdName := vd.Name
	vdKey := types.NamespacedName{Namespace: vd.Namespace, Name: vdName}

	existing := &star.VectorData{}
	getErr := r.Get(ctx, vdKey, existing)
	switch {
	case getErr == nil:
		// Already created. Record the back-reference on every successful reconcile so a status view of the
		// VectorDeployment always names the VectorData we own; cheap and idempotent.
		vd.Status.ResultingVectorData = &star.LocalObjectReference{Name: existing.Name}
		setVectorDataCreatedCondition(vd, metav1.ConditionTrue, "VectorDataCreated",
			fmt.Sprintf("VectorData %s already present", vdName))
		return nil
	case !apierrors.IsNotFound(getErr):
		setVectorDataCreatedCondition(vd, metav1.ConditionFalse, "VectorDataGetFailed", getErr.Error())
		return fmt.Errorf("failed to get VectorData %s: %w", vdKey, getErr)
	}

	// Lazy re-fetch path: if the in-memory blob from the initial reconcile is no longer in scope (later reconciles
	// where Reconcile took the cached `Status.ResolvedVectorOcm` shortcut), pull the descriptor again. Cheap on the
	// happy path because we only enter this branch when the VectorData is genuinely missing.
	configBlob := freshConfig
	if configBlob == nil && vd.Status.ResolvedVectorOcm != "" {
		descr, err := r.OcmAdapter.GetVectorDescriptor(ctx, vectorRef)
		if err != nil {
			setVectorDataCreatedCondition(vd, metav1.ConditionFalse, "ConfigBlobRefetchFailed", err.Error())
			return fmt.Errorf("failed to re-fetch vector config blob from OCM for %s: %w", vdKey, err)
		}
		configBlob = descr.Configuration
	}

	// Defensive validation: authored bytes must parse as JSON. The constraint exists because every known consumer
	// (current and planned) treats the payload as JSON. We surface it here rather than during OCM fetch so the
	// failure has the right scope ("can't produce VectorData") and so the freshConfig case (where the bytes are
	// just-fetched) is covered uniformly with the re-fetch case.
	if len(configBlob) > 0 && !json.Valid(configBlob) {
		setVectorDataCreatedCondition(vd, metav1.ConditionFalse, "InvalidConfigPayload",
			"vector configuration payload is not valid JSON")
		return fmt.Errorf("vector configuration payload is not valid JSON")
	}

	desired := &star.VectorData{
		ObjectMeta: metav1.ObjectMeta{
			Name:      vdName,
			Namespace: vd.Namespace,
		},
		Spec: star.VectorDataSpec{
			Config:            configBlob,
			DeploymentResults: vd.Status.DeploymentResults,
		},
	}

	if err := controllerutil.SetControllerReference(vd, desired, r.Scheme); err != nil {
		setVectorDataCreatedCondition(vd, metav1.ConditionFalse, "OwnerRefFailed", err.Error())
		return fmt.Errorf("failed to set controller reference on VectorData %s: %w", vdKey, err)
	}

	if err := r.Create(ctx, desired); err != nil {
		setVectorDataCreatedCondition(vd, metav1.ConditionFalse, "VectorDataCreateFailed", err.Error())
		return fmt.Errorf("failed to create VectorData %s: %w", vdKey, err)
	}

	vd.Status.ResultingVectorData = &star.LocalObjectReference{Name: desired.Name}
	setVectorDataCreatedCondition(vd, metav1.ConditionTrue, "VectorDataCreated",
		fmt.Sprintf("VectorData %s created", vdName))
	r.Recorder.Eventf(vd, nil, corev1.EventTypeNormal,
		"VectorDataCreated", "VectorDataCreated",
		fmt.Sprintf("Created VectorData %s in namespace %s", vdName, vd.Namespace))
	log.Info("VectorData created", "vectorData", vdKey.String())
	return nil
}

// vectorDataIsReady reports whether the implementor has flipped the VectorData's Ready condition. Returns true on a
// missing VectorData when the VD already reported VectorDataCreated=False (so the caller can keep the lifecycle
// blocked rather than racing with itself).
func (r *VectorDeploymentReconciler) vectorDataIsReady(ctx context.Context, vd *star.VectorDeployment) (bool, error) {
	if vd.Status.ResultingVectorData == nil {
		return false, nil
	}
	current := &star.VectorData{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: vd.Namespace, Name: vd.Status.ResultingVectorData.Name}, current); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return meta.IsStatusConditionTrue(current.Status.Conditions, star.VectorDataReadyCondition), nil
}

// setVectorDataCreatedCondition writes the VectorDataCreated condition on the parent VectorDeployment.
func setVectorDataCreatedCondition(vd *star.VectorDeployment, status metav1.ConditionStatus, reason, message string) {
	meta.SetStatusCondition(&vd.Status.Conditions, metav1.Condition{
		Type:               star.VectorDataCreatedCondition,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: vd.Generation,
		LastTransitionTime: metav1.Now(),
	})
}

// handleVectorDeploymentDeletion drives the teardown sequence when the VectorDeployment is being deleted. It deletes
// the owned VectorData CR explicitly (in addition to the controller-owner-reference cascade that Kubernetes performs
// in the background) and then removes the VectorDataFinalizer so that the API server can finalize the VD object. The
// runtime-specific implementor is in turn responsible for cleaning up its own state when the VectorData disappears.
//
// The function is idempotent: if the VectorData is already gone, the delete is a no-op; if the finalizer is already
// absent, no patch is issued.
func (r *VectorDeploymentReconciler) handleVectorDeploymentDeletion(
	ctx context.Context,
	vd *star.VectorDeployment,
	log logr.Logger,
) (ctrl.Result, error) {
	if !controllerutil.ContainsFinalizer(vd, star.VectorDataFinalizer) {
		// Nothing left for us to do; the API server will finish removing the object.
		return ctrl.Result{}, nil
	}

	vdKey := types.NamespacedName{Namespace: vd.Namespace, Name: vd.Name}

	vectorData := &star.VectorData{
		ObjectMeta: metav1.ObjectMeta{Name: vd.Name, Namespace: vd.Namespace},
	}
	if err := r.Delete(ctx, vectorData); err != nil && !apierrors.IsNotFound(err) {
		return ctrl.Result{}, fmt.Errorf("failed to delete VectorData %s during teardown: %w", vdKey, err)
	}
	log.Info("VectorData removed during VectorDeployment teardown", "vectorData", vdKey.String())

	patch := client.MergeFrom(vd.DeepCopy())
	controllerutil.RemoveFinalizer(vd, star.VectorDataFinalizer)
	if err := r.Patch(ctx, vd, patch); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to remove vector-data finalizer from %s: %w", vdKey, err)
	}
	return ctrl.Result{}, nil
}

// sha256Hex returns the hex-encoded SHA-256 of the input. Used only by tests and for tracing/logging.
func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

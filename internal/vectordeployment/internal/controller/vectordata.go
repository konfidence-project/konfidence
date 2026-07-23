package controller

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/pkg/jsonschema"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// handleVectorData creates the VectorData CR carrying the OCM-resolved envelope and aggregated DeploymentResults.
// The owning VectorDeployment is set as the controller-owner so K8s GC cascades teardown; no finalizer here. The
// landscape orchestrator materialises the payload for the target runtime.
//
// Caller is responsible for resolving `configBlob` (i.e. fetching from OCM) before calling. This function only
// observes-or-creates given an already-resolved blob; it has no awareness of cache or refetch semantics.
func (r *VectorDeploymentReconciler) handleVectorData(
	ctx context.Context,
	vd *konfidence.VectorDeployment,
	configBlob []byte,
	log logr.Logger,
) error {
	key := types.NamespacedName{Namespace: vd.Namespace, Name: vd.Name}

	existing := &konfidence.VectorData{}
	switch err := r.Get(ctx, key, existing); {
	case err == nil:
		vd.Status.ResultingVectorData = &konfidence.LocalObjectReference{Name: existing.Name}
		setVectorDataCreatedCondition(vd, metav1.ConditionTrue, "VectorDataCreated",
			fmt.Sprintf("VectorData %s already present", vd.Name))
		return nil
	case !apierrors.IsNotFound(err):
		setVectorDataCreatedCondition(vd, metav1.ConditionFalse, "VectorDataGetFailed", err.Error())
		return fmt.Errorf("get VectorData %s: %w", key, err)
	}

	features, authored, err := splitEnvelope(configBlob)
	if err != nil {
		setVectorDataCreatedCondition(vd, metav1.ConditionFalse, "InvalidConfigPayload", err.Error())
		return err
	}

	desired := &konfidence.VectorData{
		ObjectMeta: metav1.ObjectMeta{Name: vd.Name, Namespace: vd.Namespace},
		Spec: konfidence.VectorDataSpec{
			Features:          features,
			Authored:          authored,
			DeploymentResults: vd.Status.DeploymentResults,
		},
	}
	if err := controllerutil.SetControllerReference(vd, desired, r.Scheme); err != nil {
		setVectorDataCreatedCondition(vd, metav1.ConditionFalse, "OwnerRefFailed", err.Error())
		return fmt.Errorf("set controller reference on VectorData %s: %w", key, err)
	}
	if err := r.Create(ctx, desired); err != nil {
		setVectorDataCreatedCondition(vd, metav1.ConditionFalse, "VectorDataCreateFailed", err.Error())
		return fmt.Errorf("create VectorData %s: %w", key, err)
	}

	vd.Status.ResultingVectorData = &konfidence.LocalObjectReference{Name: desired.Name}
	setVectorDataCreatedCondition(vd, metav1.ConditionTrue, "VectorDataCreated",
		fmt.Sprintf("VectorData %s created", vd.Name))
	r.Recorder.Eventf(vd, nil, corev1.EventTypeNormal,
		"VectorDataCreated", "VectorDataCreated",
		fmt.Sprintf("Created VectorData %s in namespace %s", vd.Name, vd.Namespace))
	log.Info("VectorData created", "vectorData", key.String())
	return nil
}

// splitEnvelope parses an OCM-resolved vector configuration envelope into structured Spec.Features and Spec.Authored.
// Uses the shared `jsonschema.VectorConfigurationV1` contract that the vector assembly process serializes. Unknown
// schemaVersion is rejected so a future v2 envelope is caught before it silently misroutes.
func splitEnvelope(blob []byte) (*runtime.RawExtension, *runtime.RawExtension, error) {
	if len(blob) == 0 {
		return nil, nil, nil
	}
	var env jsonschema.VectorConfigurationV1
	if err := json.Unmarshal(blob, &env); err != nil {
		return nil, nil, fmt.Errorf("vector config envelope is not a JSON object: %w", err)
	}
	if env.SchemaVersion != "" && env.SchemaVersion != jsonschema.VectorConfigurationV1SchemaVersion {
		return nil, nil, fmt.Errorf("unsupported vector config schemaVersion %q (want %q)",
			env.SchemaVersion, jsonschema.VectorConfigurationV1SchemaVersion)
	}
	var features, authored *runtime.RawExtension
	if len(env.Features) > 0 {
		features = &runtime.RawExtension{Raw: append([]byte(nil), env.Features...)}
	}
	if len(env.Authored) > 0 {
		authored = &runtime.RawExtension{Raw: append([]byte(nil), env.Authored...)}
	}
	return features, authored, nil
}

func (r *VectorDeploymentReconciler) vectorDataIsReady(ctx context.Context, vd *konfidence.VectorDeployment) (bool, error) {
	if vd.Status.ResultingVectorData == nil {
		return false, nil
	}
	current := &konfidence.VectorData{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: vd.Namespace, Name: vd.Status.ResultingVectorData.Name}, current); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return meta.IsStatusConditionTrue(current.Status.Conditions, konfidence.VectorDataReadyCondition), nil
}

func setVectorDataCreatedCondition(vd *konfidence.VectorDeployment, status metav1.ConditionStatus, reason, message string) {
	meta.SetStatusCondition(&vd.Status.Conditions, metav1.Condition{
		Type:               konfidence.VectorDataCreatedCondition,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: vd.Generation,
		LastTransitionTime: metav1.Now(),
	})
}

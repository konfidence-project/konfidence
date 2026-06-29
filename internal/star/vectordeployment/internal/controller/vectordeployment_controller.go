package controller

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	VectorDeploymentControllerName = "vector-deployment-controller"
	MaxLabelSize                   = 63
)

// VectorDeploymentReconciler reconciles a VectorDeployment object
type VectorDeploymentReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Recorder   events.EventRecorder
	OcmAdapter VectorOcmPort
}

// +kubebuilder:rbac:groups=star.konfidence.cloud,resources=vectordeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=star.konfidence.cloud,resources=vectordeployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=star.konfidence.cloud,resources=vectordeployments/finalizers,verbs=update
// +kubebuilder:rbac:groups=star.konfidence.cloud,resources=artifactdeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=star.konfidence.cloud,resources=vectorassignments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=star.konfidence.cloud,resources=vectordata,verbs=get;list;watch;create;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

func (r *VectorDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconciling VectorDeployment")

	vectorDeployment := &star.VectorDeployment{}
	if err := r.Get(ctx, req.NamespacedName, vectorDeployment); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	originalVectorDeployment := vectorDeployment.DeepCopy()
	patch := client.MergeFrom(originalVectorDeployment)

	vectorRef, err := compref.Parse(vectorDeployment.Spec.Vector)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to parse vector reference %s: %w", vectorDeployment.Spec.Vector, err)
	}

	var artifactRefs []compref.Ref
	// freshConfig holds the OCM-resolved authored blob from this reconcile's fetch (nil on subsequent reconciles
	// where ResolvedVectorOcm is cached). handleVectorData refetches if needed.
	var freshConfig []byte

	if vectorDeployment.Status.ResolvedVectorOcm == "" {
		fetchedVectorDescriptor, err := r.OcmAdapter.GetVectorDescriptor(ctx, *vectorRef)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to fetch vector OCM for vector deployment %s : %w", vectorDeployment.Name, err)
		}

		vectorDeployment.Status.ResolvedVectorOcm = string(fetchedVectorDescriptor.DescriptorJSON)
		freshConfig = fetchedVectorDescriptor.Configuration
		artifactRefs = fetchedVectorDescriptor.References

		meta.SetStatusCondition(
			&vectorDeployment.Status.Conditions,
			metav1.Condition{
				Type:               star.VectorDownloadedCondition,
				Status:             metav1.ConditionTrue,
				Reason:             star.VectorDownloadedCondition,
				Message:            fmt.Sprintf("Successfully downloaded vector %s from OCM repository", vectorDeployment.Spec.Vector),
				ObservedGeneration: vectorDeployment.Generation,
				LastTransitionTime: metav1.Now(),
			},
		)
	} else {
		artifactRefs, err = artifactRefsFromStatus(vectorDeployment.Status.ResolvedVectorOcm, *vectorRef)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to parse artifact refs from status for vector deployment %s: %w", vectorDeployment.Name, err)
		}
	}

	allDeploymentsReady, err := r.handleArtifactDeployments(ctx, artifactRefs, vectorDeployment, log)
	if !reflect.DeepEqual(vectorDeployment.Status, originalVectorDeployment.Status) {
		if patchError := r.Client.Status().Patch(ctx, vectorDeployment, patch); patchError != nil {
			patchErrorMessage := "unable to update vectorDeployment status"

			if err != nil {
				reconcileError := fmt.Errorf("failed to handle artifact deployments for vector deployment %s : %w", vectorDeployment.Name, err)
				return ctrl.Result{}, fmt.Errorf("%s: %w; %w", patchErrorMessage, patchError, reconcileError)
			}

			return ctrl.Result{}, fmt.Errorf("%s: %w", patchErrorMessage, patchError)
		}
	}
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to handle artifact deployments for vector deployment %s: %w", vectorDeployment.Name, err)
	}
	if !allDeploymentsReady {
		log.Info("waiting for artifact deployments to be ready")
		return ctrl.Result{}, nil
	}

	allAssignmentsReady, err := r.handleVectorAssignments(ctx, vectorDeployment, log)
	if !reflect.DeepEqual(vectorDeployment.Status, originalVectorDeployment.Status) {
		if patchError := r.Client.Status().Patch(ctx, vectorDeployment, patch); patchError != nil {
			patchErrorMessage := "unable to update vectorDeployment status"

			if err != nil {
				reconcileError := fmt.Errorf("failed to handle vector assignments for vector deployment %s : %w", vectorDeployment.Name, err)
				return ctrl.Result{}, fmt.Errorf("%s: %w; %w", patchErrorMessage, patchError, reconcileError)
			}

			return ctrl.Result{}, fmt.Errorf("%s: %w", patchErrorMessage, patchError)
		}
	}
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to handle vector assignments for vector deployment %s: %w", vectorDeployment.Name, err)
	}
	if !allAssignmentsReady {
		log.Info("waiting for vector assignments to be ready")
		return ctrl.Result{}, nil
	}

	// Create (or re-use) the VectorData CR carrying the resolved authored configuration and aggregated
	// DeploymentResults. This step is a singleton action per vector, distinct from the per-artifact VectorAssignment
	// work above. The runtime-specific implementor (e.g. the in-tree Kubernetes adapter in `internal/star/vectordata`)
	// watches VectorData and materialises it on the target runtime (e.g. as a ConfigMap).
	configBlob, err := r.resolveVectorConfigForVectorData(ctx, vectorDeployment, *vectorRef, freshConfig)
	if err != nil {
		if !reflect.DeepEqual(vectorDeployment.Status, originalVectorDeployment.Status) {
			if patchError := r.Client.Status().Patch(ctx, vectorDeployment, patch); patchError != nil {
				return ctrl.Result{}, fmt.Errorf("unable to update vectorDeployment status: %w; %w", patchError, err)
			}
		}
		return ctrl.Result{}, fmt.Errorf("resolve vector config for vector deployment %s: %w", vectorDeployment.Name, err)
	}
	if err := r.handleVectorData(ctx, vectorDeployment, configBlob, log); err != nil {
		if !reflect.DeepEqual(vectorDeployment.Status, originalVectorDeployment.Status) {
			if patchError := r.Client.Status().Patch(ctx, vectorDeployment, patch); patchError != nil {
				return ctrl.Result{}, fmt.Errorf("unable to update vectorDeployment status: %w; %w", patchError, err)
			}
		}
		return ctrl.Result{}, fmt.Errorf("failed to create VectorData for vector deployment %s: %w", vectorDeployment.Name, err)
	}

	// Gate VectorReady on the runtime-specific implementor having reported VectorData.Ready=True. If it is still
	// pending (CR just created, implementor hasn't observed it yet, or the implementor is reporting a transient
	// error), keep the VectorReady condition unset and rely on the controller's Owns(&VectorData{}) watch to retrigger
	// reconciliation when the implementor updates the VectorData status.
	vectorDataReady, vdErr := r.vectorDataIsReady(ctx, vectorDeployment)
	if vdErr != nil {
		if !reflect.DeepEqual(vectorDeployment.Status, originalVectorDeployment.Status) {
			if patchError := r.Client.Status().Patch(ctx, vectorDeployment, patch); patchError != nil {
				return ctrl.Result{}, fmt.Errorf("unable to update vectorDeployment status: %w; %w", patchError, vdErr)
			}
		}
		return ctrl.Result{}, fmt.Errorf("failed to read VectorData readiness for vector deployment %s: %w", vectorDeployment.Name, vdErr)
	}
	if !vectorDataReady {
		log.Info("waiting for VectorData to be materialized by the runtime implementor")
		if !reflect.DeepEqual(vectorDeployment.Status, originalVectorDeployment.Status) {
			if patchError := r.Client.Status().Patch(ctx, vectorDeployment, patch); patchError != nil {
				return ctrl.Result{}, fmt.Errorf("unable to update vectorDeployment status: %w", patchError)
			}
		}
		return ctrl.Result{}, nil
	}

	// set status condition VectorReadyCondition to True
	meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{
		Type:               star.VectorReadyCondition,
		Status:             metav1.ConditionTrue,
		Reason:             star.VectorReadyCondition,
		Message:            fmt.Sprintf("Vector deployment %s is ready", vectorDeployment.Name),
		ObservedGeneration: vectorDeployment.Generation,
		LastTransitionTime: metav1.Now(),
	})

	if !reflect.DeepEqual(vectorDeployment.Status, originalVectorDeployment.Status) {
		if patchError := r.Client.Status().Patch(ctx, vectorDeployment, patch); patchError != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update vectorDeployment status: %w", patchError)
		}
	}

	log.Info("VectorDeployment reconciled successfully")
	return ctrl.Result{}, nil
}

func (r *VectorDeploymentReconciler) handleArtifactDeployments(
	ctx context.Context,
	artifactReferences []compref.Ref,
	vectorDeployment *star.VectorDeployment,
	log logr.Logger,
) (bool, error) {
	// Build fresh maps from scratch so removed artifacts are no longer referenced.
	// We use nil initially and allocate lazily to avoid spurious status patches when
	// DeepEqual compares nil (server value after omitempty round-trip) vs. empty map.
	var (
		resultingArtifactDeployments = make(map[string]star.LocalArtifactDeploymentReference, len(artifactReferences))
		deploymentResults            = make(map[string]star.DeploymentResult)
	)
	allReady := true

	// TODO parallelize and handle partial failures
	for _, artifactRef := range artifactReferences {
		// fetch the artifact component version from OCI
		artifactManifest, err := r.OcmAdapter.GetArtifactManifestByReference(ctx, artifactRef)
		if err != nil {
			return false, fmt.Errorf("failed to fetch artifact component version for %q: %w", artifactRef.String(), err)
		}

		var uid *string
		if !artifactManifest.AllowReuse {
			uid = new(string(vectorDeployment.UID))
		}

		deploymentName, err := ConstructArtifactDeploymentName(artifactRef.Component, artifactRef.Version, uid)
		if err != nil {
			return false, fmt.Errorf("failed to construct artifact deployment name for %q: %w", artifactRef.String(), err)
		}

		// fetch existing artifact deployment from k8s api
		artifactDeployment := &star.ArtifactDeployment{}
		err = r.Get(ctx, types.NamespacedName{Namespace: vectorDeployment.Namespace, Name: deploymentName}, artifactDeployment)
		if err != nil {
			// if error is not NotFound then return error
			if !apierrors.IsNotFound(err) {
				return false, fmt.Errorf("failed to get artifact deployment %q: %w", deploymentName, err)
			}

			log.Info("ArtifactDeployment not found, create new one", "name", deploymentName)
			artifactDeployment = r.constructArtifactDeployment(artifactRef, artifactManifest, vectorDeployment, deploymentName)
			if err := r.Create(ctx, artifactDeployment); err != nil {
				return false, fmt.Errorf("failed to create ArtifactDeployment resource %s: %w", deploymentName, err)
			}
			msg := fmt.Sprintf("Created ArtifactDeployment %s for VectorDeployment %s", deploymentName, vectorDeployment.Name)
			r.Recorder.Eventf(vectorDeployment, nil, corev1.EventTypeNormal, "ArtifactDeploymentCreated", "ArtifactDeploymentCreated", msg)
			log.Info(msg)
		} else {
			log.Info("ArtifactDeployment found, update existing one", "name", deploymentName)
		}

		var ownerRef *metav1.OwnerReference = nil
		for _, ref := range artifactDeployment.OwnerReferences {
			if ref.UID == vectorDeployment.UID {
				ownerRef = &ref
				break
			}
		}

		if ownerRef == nil {
			log.Info("Adding owner reference to existing artifact deployment", "vector", vectorDeployment.Spec.Vector, "name", artifactDeployment.Name)
			if err := controllerutil.SetOwnerReference(vectorDeployment, artifactDeployment, r.Scheme); err != nil {
				return false, fmt.Errorf("unable to add vectorDeployment owner reference to artifactDeployment: %w", err)
			}

			if err := r.Update(ctx, artifactDeployment); err != nil {
				return false, fmt.Errorf("failed to set owner reference for ArtifactDeployment %q: %w", artifactDeployment.Name, err)
			}
			r.Recorder.Eventf(vectorDeployment, nil, corev1.EventTypeNormal,
				"ArtifactDeploymentUpdated", "ArtifactDeploymentUpdated",
				fmt.Sprintf("Updated owner reference for ArtifactDeployment %s", artifactDeployment.Name))
		} else {
			log.Info("ArtifactDeployment already has owner reference", "vector", vectorDeployment.Spec.Vector, "name", artifactDeployment.Name)
		}

		// Update the artifact deployment to the status map of the VectorDeployment
		resultingArtifactDeployments[artifactRef.Component] = star.LocalArtifactDeploymentReference{
			Name: artifactDeployment.Name,
		}

		// state management for VectorDeployedCondition
		if meta.IsStatusConditionTrue(artifactDeployment.Status.Conditions, star.DeploymentResultCreatedCondition) {
			// collect deployment results
			for _, result := range artifactDeployment.Status.DeploymentResults {
				deploymentResults[artifactRef.Component+"/"+result.Name] = result
			}
		}
		if !meta.IsStatusConditionTrue(artifactDeployment.Status.Conditions, star.ArtifactDeploymentReadyCondition) {
			allReady = false
		}
	}

	// Write local maps back to status. Assign nil when empty so that the
	// omitempty JSON tag round-trips cleanly and reflect.DeepEqual stays stable.
	if len(resultingArtifactDeployments) > 0 {
		vectorDeployment.Status.ResultingArtifactDeployments = resultingArtifactDeployments
	} else {
		vectorDeployment.Status.ResultingArtifactDeployments = nil
	}
	if len(deploymentResults) > 0 {
		vectorDeployment.Status.DeploymentResults = deploymentResults
	} else {
		vectorDeployment.Status.DeploymentResults = nil
	}

	// set status condition ArtifactDeploymentsCreatedCondition to created
	meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{
		Type:               star.ArtifactDeploymentsCreatedCondition,
		Status:             metav1.ConditionTrue,
		Reason:             star.ArtifactDeploymentsCreatedCondition,
		Message:            fmt.Sprintf("Successfully created Artifact deployments for vector deployment %s", vectorDeployment.Name),
		ObservedGeneration: vectorDeployment.Generation,
		LastTransitionTime: metav1.Now(),
	})

	if allReady {
		meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{
			Type:               star.VectorDeployedCondition,
			Status:             metav1.ConditionTrue,
			Reason:             star.VectorDeployedCondition,
			Message:            fmt.Sprintf("All artifacts of vector deployment %s are deployed", vectorDeployment.Name),
			ObservedGeneration: vectorDeployment.Generation,
			LastTransitionTime: metav1.Now(),
		})
	}

	return allReady, nil
}

func (r *VectorDeploymentReconciler) handleVectorAssignments(ctx context.Context, vectorDeployment *star.VectorDeployment, log logr.Logger) (bool, error) {
	resultingVectorAssignments := make(map[string]star.LocalVectorAssignmentReference, len(vectorDeployment.Status.ResultingArtifactDeployments))
	allReady := true

	for componentName, artifactDeployment := range vectorDeployment.Status.ResultingArtifactDeployments {
		// fetch existing artifact assignment from k8s api
		vectorAssignment := &star.VectorAssignment{}
		assignmentName := vectorDeployment.Name
		err := r.Get(ctx, types.NamespacedName{Namespace: vectorDeployment.Namespace, Name: assignmentName}, vectorAssignment)
		if err != nil {
			// if error is not NotFound then return error
			if !apierrors.IsNotFound(err) {
				return false, fmt.Errorf("failed to get vector assignment %q: %w", assignmentName, err)
			}

			// create a new VectorAssignment
			ad := &star.ArtifactDeployment{}
			err = r.Get(ctx, types.NamespacedName{Namespace: vectorDeployment.Namespace, Name: artifactDeployment.Name}, ad)
			if err != nil {
				return false, fmt.Errorf("failed to get artifact deployment %q for vector assignment %q: %w", artifactDeployment.Name, assignmentName, err)
			}

			log.Info("VectorAssignment not found, create new one", "name", assignmentName)

			vectorAssignment = &star.VectorAssignment{
				ObjectMeta: ctrl.ObjectMeta{
					Name:      assignmentName,
					Namespace: vectorDeployment.Namespace,
					Labels: map[string]string{
						pkgctrl.ArtifactReferenceLabel: artifactDeployment.Name,
					},
				},
				Spec: star.VectorAssignmentSpec{
					Manifest:              ad.Spec.Manifest,
					ArtifactDeploymentRef: artifactDeployment,
					VectorDeploymentRef: star.LocalVectorDeploymentReference{
						Name: vectorDeployment.Name,
					},
				},
			}
			if err := r.Create(ctx, vectorAssignment); err != nil {
				return false, fmt.Errorf("failed to create VectorAssignment %q: %w", assignmentName, err)
			}
			msg := fmt.Sprintf("Created VectorAssignment %s", assignmentName)
			r.Recorder.Eventf(vectorDeployment, nil, corev1.EventTypeNormal, "VectorAssignmentCreated", "VectorAssignmentCreated", msg)
			log.Info(msg)
		} else {
			log.Info("VectorAssignment found, update existing one", "name", assignmentName)
		}

		var ownerRef *metav1.OwnerReference = nil
		for _, ref := range vectorAssignment.OwnerReferences {
			if ref.UID == vectorDeployment.UID {
				ownerRef = &ref
				break
			}
		}

		if ownerRef == nil {
			log.Info("Adding owner reference to existing vector assignment", "vector", vectorDeployment.Spec.Vector, "name", vectorAssignment.Name)
			if err := controllerutil.SetControllerReference(vectorDeployment, vectorAssignment, r.Scheme); err != nil {
				return false, fmt.Errorf("unable to add vectorDeployment owner reference to vectorAssignment: %w", err)
			}

			if err := r.Update(ctx, vectorAssignment); err != nil {
				return false, fmt.Errorf("failed to set owner reference for VectorAssignment %q: %w", vectorAssignment.Name, err)
			}
			r.Recorder.Eventf(vectorDeployment, nil, corev1.EventTypeNormal,
				"VectorAssignmentUpdated", "VectorAssignmentUpdated",
				fmt.Sprintf("Updated owner reference for VectorAssignment %s", vectorAssignment.Name))
		} else {
			log.Info("Vector deployment already has owner reference", "vector", vectorDeployment.Spec.Vector, "name", vectorAssignment.Name)
		}

		// Update the artifact assignment to the status map of the VectorDeployment
		resultingVectorAssignments[componentName] = star.LocalVectorAssignmentReference{
			Name: vectorAssignment.Name,
		}

		// state management for VectorAssignmentsCreatedCondition
		if !meta.IsStatusConditionTrue(vectorAssignment.Status.Conditions, star.VectorReadyCondition) {
			allReady = false
		}
	}

	// Write local map back to status. Assign nil when empty so that the
	// omitempty JSON tag round-trips cleanly and reflect.DeepEqual stays stable.
	if len(resultingVectorAssignments) > 0 {
		vectorDeployment.Status.ResultingVectorAssignments = resultingVectorAssignments
	} else {
		vectorDeployment.Status.ResultingVectorAssignments = nil
	}

	// set status condition VectorAssignmentsCreatedCondition. VectorReady is intentionally NOT set here any more
	// — it now exclusively flips after VectorData has been materialised by the runtime-specific orchestrator,
	// see the post-handleVectorData block in Reconcile().
	meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{
		Type:               star.VectorAssignmentsCreatedCondition,
		Status:             metav1.ConditionTrue,
		Reason:             star.VectorAssignmentsCreatedCondition,
		Message:            fmt.Sprintf("Successfully created vector assignments for vector deployment %s", vectorDeployment.Name),
		ObservedGeneration: vectorDeployment.Generation,
		LastTransitionTime: metav1.Now(),
	})

	return allReady, nil
}

func mapTaskManifestsToLandscape(taskManifests []TaskManifest) []star.TaskManifest {
	landscapeTaskManifests := make([]star.TaskManifest, len(taskManifests))
	for i, taskManifest := range taskManifests {
		landscapeTaskManifests[i] = star.TaskManifest{
			Name:      taskManifest.Name,
			Type:      taskManifest.Type,
			DependsOn: taskManifest.DependsOn,
			Spec:      runtime.RawExtension{Raw: []byte(taskManifest.Spec)},
		}
	}
	return landscapeTaskManifests
}

func mapArtifactResourcesToLandscape(resources []OCMResource) []star.OCMResource {
	landscapeResources := make([]star.OCMResource, 0, len(resources))
	for _, resource := range resources {
		landscapeResources = append(landscapeResources, star.OCMResource{
			Name:    resource.Name,
			Content: runtime.RawExtension{Raw: resource.Content},
			Type:    resource.Type,
		})
	}
	return landscapeResources
}

// SetupWithManager sets up the controller with the Manager.
func (r *VectorDeploymentReconciler) SetupWithManager(mgr ctrl.Manager, controllerName string) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&star.VectorDeployment{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&star.ArtifactDeployment{}, builder.MatchEveryOwner).
		Owns(&star.VectorAssignment{}, builder.MatchEveryOwner).
		// Re-reconcile the parent VectorDeployment when the runtime-specific implementor flips
		// VectorData.Status.Ready, so the lifecycle can progress to VectorReady without polling.
		Owns(&star.VectorData{}).
		Named(controllerName).
		Complete(r)
}

func (r *VectorDeploymentReconciler) constructArtifactDeployment(
	ref compref.Ref,
	artifactManifest ArtifactManifest,
	vectorDeployment *star.VectorDeployment,
	deploymentName string,
) *star.ArtifactDeployment {
	// map task manifests from domain.TaskManifest to star.TaskManifest
	taskManifests := mapTaskManifestsToLandscape(artifactManifest.Tasks)
	artifactResources := mapArtifactResourcesToLandscape(artifactManifest.Resources)
	return &star.ArtifactDeployment{
		ObjectMeta: ctrl.ObjectMeta{
			Name:      deploymentName,
			Namespace: vectorDeployment.Namespace,
		},
		Spec: star.ArtifactDeploymentSpec{
			Manifest: star.ArtifactManifest{
				Type:       artifactManifest.Type,
				AllowReuse: artifactManifest.AllowReuse,
			},
			TaskManifests: taskManifests,
			Component: star.OCMComponent{
				Name:      ref.Component,
				Version:   ref.Version,
				Resources: artifactResources,
			},
		},
	}
}

// resolveVectorConfigForVectorData returns the OCM-resolved vector config blob to hand to handleVectorData.
// On the first reconcile freshConfig is already in scope; on later reconciles where the VectorData CR is missing
// (deleted out-of-band, fresh cluster after status was cached, etc.) we refetch the descriptor so handleVectorData
// only ever sees an already-resolved blob and has no awareness of cache/refetch semantics. May return nil if the
// VectorData already exists (handleVectorData ignores the input in that case).
func (r *VectorDeploymentReconciler) resolveVectorConfigForVectorData(
	ctx context.Context,
	vd *star.VectorDeployment,
	vectorRef compref.Ref,
	freshConfig []byte,
) ([]byte, error) {
	if freshConfig != nil {
		return freshConfig, nil
	}
	vdKey := types.NamespacedName{Name: vd.Name, Namespace: vd.Namespace}
	if err := r.Get(ctx, vdKey, &star.VectorData{}); err == nil {
		return nil, nil
	} else if !apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("get VectorData %s: %w", vdKey, err)
	}
	descr, err := r.OcmAdapter.GetVectorDescriptor(ctx, vectorRef)
	if err != nil {
		return nil, fmt.Errorf("refetch OCM config blob: %w", err)
	}
	return descr.Configuration, nil
}

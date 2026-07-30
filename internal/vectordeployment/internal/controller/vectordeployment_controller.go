package controller

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/go-logr/logr"
	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	pkgctrl "github.com/konfidence-project/konfidence/pkg/controller"
	"github.com/konfidence-project/konfidence/pkg/hash"
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

// errArtifactDeploymentCollision signals that a deterministic ArtifactDeployment name collided with a
// different artifact and its collisionCount salt was bumped. It is not a reconcile failure: the caller
// persists the bumped salt and requeues cleanly (no error backoff) so the next reconcile deploys under
// the re-salted name.
var errArtifactDeploymentCollision = errors.New("artifact deployment name collision; salt bumped, requeueing")

// VectorDeploymentReconciler reconciles a VectorDeployment object
type VectorDeploymentReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Recorder   events.EventRecorder
	OcmAdapter VectorOcmPort
}

type resolvedVector struct {
	descriptorJSON []byte
	artifactRefs   []compref.Ref
	config         []byte
	configResolved bool
}

// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectordeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectordeployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectordeployments/finalizers,verbs=update
// +kubebuilder:rbac:groups=konfidence.cloud,resources=artifactdeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectorassignments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectordata,verbs=get;list;watch;create;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

func (r *VectorDeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) { //nolint:gocyclo
	log := logf.FromContext(ctx)
	log.Info("Reconciling VectorDeployment")

	vectorDeployment := &konfidence.VectorDeployment{}
	if err := r.Get(ctx, req.NamespacedName, vectorDeployment); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	originalVectorDeployment := vectorDeployment.DeepCopy()
	patch := client.MergeFrom(originalVectorDeployment)

	vectorRef, err := compref.Parse(vectorDeployment.Spec.Vector)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to parse vector reference %s: %w", vectorDeployment.Spec.Vector, err)
	}

	resolvedVector, err := r.resolveVector(ctx, vectorDeployment, *vectorRef)
	if err != nil {
		return ctrl.Result{}, err
	}

	if len(resolvedVector.descriptorJSON) > 0 {
		vectorDeployment.Status.ResolvedVectorOcm = string(resolvedVector.descriptorJSON)
		meta.SetStatusCondition(
			&vectorDeployment.Status.Conditions,
			metav1.Condition{
				Type:               konfidence.VectorDownloadedCondition,
				Status:             metav1.ConditionTrue,
				Reason:             konfidence.VectorDownloadedCondition,
				Message:            fmt.Sprintf("Successfully downloaded vector %s from OCM repository", vectorDeployment.Spec.Vector),
				ObservedGeneration: vectorDeployment.Generation,
				LastTransitionTime: metav1.Now(),
			},
		)
	}

	allDeploymentsReady, err := r.handleArtifactDeployments(ctx, resolvedVector.artifactRefs, vectorDeployment, log)
	if !reflect.DeepEqual(vectorDeployment.Status, originalVectorDeployment.Status) {
		if patchError := r.Client.Status().Patch(ctx, vectorDeployment, patch); patchError != nil {
			patchErrorMessage := "unable to update vectorDeployment status"

			if err != nil && !errors.Is(err, errArtifactDeploymentCollision) {
				reconcileError := fmt.Errorf("failed to handle artifact deployments for vector deployment %s : %w", vectorDeployment.Name, err)
				return ctrl.Result{}, fmt.Errorf("%s: %w; %w", patchErrorMessage, patchError, reconcileError)
			}

			return ctrl.Result{}, fmt.Errorf("%s: %w", patchErrorMessage, patchError)
		}
	}
	if err != nil {
		// A name collision is not a failure: the bumped salt was just persisted above, so requeue cleanly
		// (no error backoff) to deploy under the re-salted name on the next reconcile.
		if errors.Is(err, errArtifactDeploymentCollision) {
			return ctrl.Result{RequeueAfter: time.Second}, nil
		}
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
	// work above. The runtime-specific implementor (e.g. the in-tree Kubernetes adapter in `internal/vectordata`)
	// watches VectorData and materialises it on the target runtime (e.g. as a ConfigMap).
	resolvedVector, err = r.ensureVectorDataConfigResolved(ctx, vectorDeployment, *vectorRef, resolvedVector)
	if err != nil {
		if !reflect.DeepEqual(vectorDeployment.Status, originalVectorDeployment.Status) {
			if patchError := r.Client.Status().Patch(ctx, vectorDeployment, patch); patchError != nil {
				return ctrl.Result{}, fmt.Errorf("unable to update vectorDeployment status: %w; %w", patchError, err)
			}
		}
		return ctrl.Result{}, fmt.Errorf("resolve vector config for vector deployment %s: %w", vectorDeployment.Name, err)
	}
	if err := r.handleVectorData(ctx, vectorDeployment, resolvedVector.config, log); err != nil {
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
		Type:               konfidence.VectorReadyCondition,
		Status:             metav1.ConditionTrue,
		Reason:             konfidence.VectorReadyCondition,
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
	vectorDeployment *konfidence.VectorDeployment,
	log logr.Logger,
) (bool, error) {
	// Build fresh maps from scratch so removed artifacts are no longer referenced.
	// We use nil initially and allocate lazily to avoid spurious status patches when
	// DeepEqual compares nil (server value after omitempty round-trip) vs. empty map.
	var (
		resultingArtifactDeployments = make(map[string]konfidence.LocalArtifactDeploymentReference, len(artifactReferences))
		deploymentResults            = make(map[string]konfidence.DeploymentResult)
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

		// Read the persisted collision salt for this artifact (keyed by component). The loop builds a fresh
		// local map and only writes it back after the range, so this reads last reconcile's value. nil -> 0.
		prev := vectorDeployment.Status.ResultingArtifactDeployments[artifactRef.Component]
		var collisionCount int32
		if prev.CollisionCount != nil {
			collisionCount = *prev.CollisionCount
		}

		deploymentName, deploymentHash, err := ConstructArtifactDeploymentName(artifactRef.Component, artifactRef.Version, uid, collisionCount)
		if err != nil {
			return false, fmt.Errorf("failed to construct artifact deployment name for %q: %w", artifactRef.String(), err)
		}

		// fetch existing artifact deployment from k8s api
		artifactDeployment := &konfidence.ArtifactDeployment{}
		err = r.Get(ctx, types.NamespacedName{Namespace: vectorDeployment.Namespace, Name: deploymentName}, artifactDeployment)
		if err != nil {
			// if error is not NotFound then return error
			if !apierrors.IsNotFound(err) {
				return false, fmt.Errorf("failed to get artifact deployment %q: %w", deploymentName, err)
			}

			log.Info("ArtifactDeployment not found, create new one", "name", deploymentName)
			artifactDeployment = r.constructArtifactDeployment(artifactRef, artifactManifest, vectorDeployment, deploymentName, deploymentHash, uid)
			if err := r.Create(ctx, artifactDeployment); err != nil {
				return false, fmt.Errorf("failed to create ArtifactDeployment resource %s: %w", deploymentName, err)
			}
			msg := fmt.Sprintf("Created ArtifactDeployment %s for VectorDeployment %s", deploymentName, vectorDeployment.Name)
			r.Recorder.Eventf(vectorDeployment, nil, corev1.EventTypeNormal, "ArtifactDeploymentCreated", "ArtifactDeploymentCreated", msg)
			log.Info(msg)
		} else {
			// A deterministic name collision: the AD found under this name belongs to a different artifact.
			// Recover the K8s Deployment way -- bump the per-artifact collisionCount salt, persist it, and
			// requeue so the next reconcile computes a fresh (re-salted) name. Not a hard failure.
			gotComponent := artifactDeployment.Annotations[pkgctrl.ArtifactComponentAnnotation]
			gotVersion := artifactDeployment.Annotations[pkgctrl.ArtifactVersionAnnotation]
			if gotComponent != artifactRef.Component || gotVersion != artifactRef.Version {
				msg := fmt.Sprintf(
					"ArtifactDeployment name %q collides: expected component %q version %q but found component %q version %q",
					deploymentName, artifactRef.Component, artifactRef.Version, gotComponent, gotVersion)
				r.Recorder.Eventf(vectorDeployment, nil, corev1.EventTypeWarning,
					"ArtifactDeploymentHashCollision", "ArtifactDeploymentHashCollision", msg)

				// Guard against an unbounded bump loop: this many consecutive salts still colliding is a bug,
				// not bad luck in a large hash space, so fail loudly instead of requeueing forever.
				if collisionCount >= 5 {
					log.Error(nil, msg, "name", deploymentName, "collisionCount", collisionCount)
					return false, fmt.Errorf("%s (giving up after %d salts)", msg, collisionCount)
				}

				collisionCount++
				log.Info("ArtifactDeployment name collision, bumping salt and requeueing",
					"name", deploymentName, "component", artifactRef.Component, "collisionCount", collisionCount)
				// Persist only the bumped salt for this component and requeue. We write straight to status
				// (rather than the loop-local map) because we return before the post-range write-back runs.
				// Preserve any existing entries; the next reconcile rebuilds the full map from scratch.
				if vectorDeployment.Status.ResultingArtifactDeployments == nil {
					vectorDeployment.Status.ResultingArtifactDeployments = map[string]konfidence.LocalArtifactDeploymentReference{}
				}
				vectorDeployment.Status.ResultingArtifactDeployments[artifactRef.Component] = konfidence.LocalArtifactDeploymentReference{
					Name:           prev.Name,
					CollisionCount: &collisionCount,
				}
				return false, errArtifactDeploymentCollision
			}
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

		// Update the artifact deployment to the status map of the VectorDeployment. Carry the collision salt
		// forward: once bumped it is permanent for this artifact slot, otherwise the next reconcile would
		// recompute the unsalted (colliding) name and orphan this deployment.
		resultingArtifactDeployments[artifactRef.Component] = konfidence.LocalArtifactDeploymentReference{
			Name:           artifactDeployment.Name,
			CollisionCount: prev.CollisionCount,
		}

		// state management for VectorDeployedCondition
		if meta.IsStatusConditionTrue(artifactDeployment.Status.Conditions, konfidence.DeploymentResultCreatedCondition) {
			// collect deployment results
			for _, result := range artifactDeployment.Status.DeploymentResults {
				deploymentResults[artifactRef.Component+"/"+result.Name] = result
			}
		}
		if !meta.IsStatusConditionTrue(artifactDeployment.Status.Conditions, konfidence.ArtifactDeploymentReadyCondition) {
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
		Type:               konfidence.ArtifactDeploymentsCreatedCondition,
		Status:             metav1.ConditionTrue,
		Reason:             konfidence.ArtifactDeploymentsCreatedCondition,
		Message:            fmt.Sprintf("Successfully created Artifact deployments for vector deployment %s", vectorDeployment.Name),
		ObservedGeneration: vectorDeployment.Generation,
		LastTransitionTime: metav1.Now(),
	})

	if allReady {
		meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{
			Type:               konfidence.VectorDeployedCondition,
			Status:             metav1.ConditionTrue,
			Reason:             konfidence.VectorDeployedCondition,
			Message:            fmt.Sprintf("All artifacts of vector deployment %s are deployed", vectorDeployment.Name),
			ObservedGeneration: vectorDeployment.Generation,
			LastTransitionTime: metav1.Now(),
		})
	}

	return allReady, nil
}

func (r *VectorDeploymentReconciler) handleVectorAssignments(
	ctx context.Context, vectorDeployment *konfidence.VectorDeployment, log logr.Logger,
) (bool, error) {
	resultingVectorAssignments := make(map[string]konfidence.LocalVectorAssignmentReference, len(vectorDeployment.Status.ResultingArtifactDeployments))
	allReady := true

	for componentName, artifactDeployment := range vectorDeployment.Status.ResultingArtifactDeployments {
		// fetch existing artifact assignment from k8s api
		vectorAssignment := &konfidence.VectorAssignment{}
		assignmentName := constructVectorAssignmentName(vectorDeployment.Name, artifactDeployment.Name)
		err := r.Get(ctx, types.NamespacedName{Namespace: vectorDeployment.Namespace, Name: assignmentName}, vectorAssignment)
		if err != nil {
			// if error is not NotFound then return error
			if !apierrors.IsNotFound(err) {
				return false, fmt.Errorf("failed to get vector assignment %q: %w", assignmentName, err)
			}

			// create a new VectorAssignment
			ad := &konfidence.ArtifactDeployment{}
			err = r.Get(ctx, types.NamespacedName{Namespace: vectorDeployment.Namespace, Name: artifactDeployment.Name}, ad)
			if err != nil {
				return false, fmt.Errorf("failed to get artifact deployment %q for vector assignment %q: %w", artifactDeployment.Name, assignmentName, err)
			}

			log.Info("VectorAssignment not found, create new one", "name", assignmentName)

			vectorAssignment = &konfidence.VectorAssignment{
				ObjectMeta: ctrl.ObjectMeta{
					Name:      assignmentName,
					Namespace: vectorDeployment.Namespace,
					Labels: map[string]string{
						pkgctrl.ArtifactReferenceLabel: artifactDeployment.Name,
					},
				},
				Spec: konfidence.VectorAssignmentSpec{
					Manifest:              ad.Spec.Manifest,
					ArtifactDeploymentRef: artifactDeployment,
					VectorDeploymentRef: konfidence.LocalVectorDeploymentReference{
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
		resultingVectorAssignments[componentName] = konfidence.LocalVectorAssignmentReference{
			Name: vectorAssignment.Name,
		}

		// state management for VectorAssignmentsCreatedCondition
		if !meta.IsStatusConditionTrue(vectorAssignment.Status.Conditions, konfidence.VectorReadyCondition) {
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
		Type:               konfidence.VectorAssignmentsCreatedCondition,
		Status:             metav1.ConditionTrue,
		Reason:             konfidence.VectorAssignmentsCreatedCondition,
		Message:            fmt.Sprintf("Successfully created vector assignments for vector deployment %s", vectorDeployment.Name),
		ObservedGeneration: vectorDeployment.Generation,
		LastTransitionTime: metav1.Now(),
	})

	return allReady, nil
}

func mapTaskManifestsToLandscape(taskManifests []TaskManifest) []konfidence.TaskManifest {
	landscapeTaskManifests := make([]konfidence.TaskManifest, len(taskManifests))
	for i, taskManifest := range taskManifests {
		landscapeTaskManifests[i] = konfidence.TaskManifest{
			Name:      taskManifest.Name,
			Type:      taskManifest.Type,
			DependsOn: taskManifest.DependsOn,
			Spec:      runtime.RawExtension{Raw: []byte(taskManifest.Spec)},
		}
	}
	return landscapeTaskManifests
}

func mapArtifactResourcesToLandscape(resources []OCMResource) []konfidence.OCMResource {
	landscapeResources := make([]konfidence.OCMResource, 0, len(resources))
	for _, resource := range resources {
		landscapeResources = append(landscapeResources, konfidence.OCMResource{
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
		For(&konfidence.VectorDeployment{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&konfidence.ArtifactDeployment{}, builder.MatchEveryOwner).
		Owns(&konfidence.VectorAssignment{}, builder.MatchEveryOwner).
		// Re-reconcile the parent VectorDeployment when the runtime-specific implementor flips
		// VectorData.Status.Ready, so the lifecycle can progress to VectorReady without polling.
		Owns(&konfidence.VectorData{}).
		Named(controllerName).
		Complete(r)
}

func (r *VectorDeploymentReconciler) constructArtifactDeployment(
	ref compref.Ref,
	artifactManifest ArtifactManifest,
	vectorDeployment *konfidence.VectorDeployment,
	deploymentName string,
	deploymentHash string,
	uid *string,
) *konfidence.ArtifactDeployment {
	// map task manifests from domain.TaskManifest to konfidence.TaskManifest
	taskManifests := mapTaskManifestsToLandscape(artifactManifest.Tasks)
	artifactResources := mapArtifactResourcesToLandscape(artifactManifest.Resources)

	ann := map[string]string{
		pkgctrl.ArtifactComponentAnnotation: ref.Component,
		pkgctrl.ArtifactVersionAnnotation:   ref.Version,
		pkgctrl.ArtifactHashAnnotation:      deploymentHash,
		pkgctrl.AllowReuseAnnotation:        fmt.Sprintf("%t", artifactManifest.AllowReuse),
	}
	if uid != nil {
		ann[pkgctrl.VectorDeploymentUIDAnnotation] = *uid
	}

	return &konfidence.ArtifactDeployment{
		ObjectMeta: ctrl.ObjectMeta{
			Name:        deploymentName,
			Namespace:   vectorDeployment.Namespace,
			Annotations: ann,
		},
		Spec: konfidence.ArtifactDeploymentSpec{
			Manifest: konfidence.ArtifactManifest{
				Type:       artifactManifest.Type,
				AllowReuse: artifactManifest.AllowReuse,
			},
			TaskManifests: taskManifests,
			Component: konfidence.OCMComponent{
				Name:      ref.Component,
				Version:   ref.Version,
				Resources: artifactResources,
			},
		},
	}
}

// resolveVector returns the vector data needed by the reconcile flow. Artifact references can be reconstructed from
// the cached descriptor status, but the config blob is only available when the descriptor is fetched from OCM.
func (r *VectorDeploymentReconciler) resolveVector(
	ctx context.Context,
	vd *konfidence.VectorDeployment,
	vectorRef compref.Ref,
) (resolvedVector, error) {
	if vd.Status.ResolvedVectorOcm == "" {
		descr, err := r.OcmAdapter.GetVectorDescriptor(ctx, vectorRef)
		if err != nil {
			return resolvedVector{}, fmt.Errorf("failed to fetch vector OCM for vector deployment %s : %w", vd.Name, err)
		}
		return resolvedVector{
			descriptorJSON: descr.DescriptorJSON,
			artifactRefs:   descr.References,
			config:         descr.Configuration,
			configResolved: true,
		}, nil
	}

	artifactRefs, err := artifactRefsFromStatus(vd.Status.ResolvedVectorOcm, vectorRef)
	if err != nil {
		return resolvedVector{}, fmt.Errorf("failed to parse artifact refs from status for vector deployment %s: %w", vd.Name, err)
	}
	return resolvedVector{artifactRefs: artifactRefs}, nil
}

func (r *VectorDeploymentReconciler) ensureVectorDataConfigResolved(
	ctx context.Context,
	vd *konfidence.VectorDeployment,
	vectorRef compref.Ref,
	resolved resolvedVector,
) (resolvedVector, error) {
	if resolved.configResolved {
		return resolved, nil
	}
	// Cheap kube-apiserver read to avoid the more expensive OCM refetch: if the VectorData CR already exists,
	// handleVectorData only observes it and never needs the config blob, so skip resolution entirely.
	if err := r.Get(ctx, types.NamespacedName{Name: vd.Name, Namespace: vd.Namespace}, &konfidence.VectorData{}); err == nil {
		resolved.configResolved = true
		return resolved, nil
	} else if !apierrors.IsNotFound(err) {
		return resolvedVector{}, fmt.Errorf("get VectorData %s/%s: %w", vd.Namespace, vd.Name, err)
	}
	descr, err := r.OcmAdapter.GetVectorDescriptor(ctx, vectorRef)
	if err != nil {
		return resolvedVector{}, fmt.Errorf("refetch OCM config blob: %w", err)
	}
	resolved.config = descr.Configuration
	resolved.configResolved = true
	return resolved, nil
}

func constructVectorAssignmentName(vectorName string, artifactName string) string {
	h := hash.Fnv(fmt.Sprintf("%s-%s", vectorName, artifactName), 13)
	return fmt.Sprintf("%s-%s", vectorName, h)
}

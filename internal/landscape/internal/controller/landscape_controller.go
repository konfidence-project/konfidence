package controller

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	defaultReconcileInterval = 30 * time.Second
	deletionRequeueInterval  = 5 * time.Second
	landscapeControllerName  = "landscape-controller"

	// The landscape namespace must be fully deleted together with everything in
	// it before the Landscape is released. The finalizer lets the controller
	// delete the namespace and wait for its termination first.
	landscapeFinalizer = "konfidence.cloud/landscape-finalizer"

	// landscapeNamespaceTypeValue is the type label value on landscape namespaces.
	landscapeNamespaceTypeValue = "landscape"

	// Event reasons for namespace lifecycle transitions.
	namespaceCreatedEventReason = "NamespaceCreated"
	namespaceUpdatedEventReason = "NamespaceUpdated"
	namespaceDeletedEventReason = "NamespaceDeleted"
)

var (
	errNamespaceConflict    = errors.New("namespace conflict")
	errNamespaceTerminating = errors.New("namespace terminating")
)

// LandscapeReconciler reconciles a Landscape object
type LandscapeReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder events.EventRecorder
}

// +kubebuilder:rbac:groups=konfidence.cloud,resources=landscapes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=konfidence.cloud,resources=landscapes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=konfidence.cloud,resources=landscapes/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=events.k8s.io,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *LandscapeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconcile landscape started...")

	landscape := &konfidence.Landscape{}
	if err := r.Get(ctx, req.NamespacedName, landscape); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !landscape.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, landscape)
	}

	if controllerutil.AddFinalizer(landscape, landscapeFinalizer) {
		if err := r.Update(ctx, landscape); err != nil {
			return ctrl.Result{}, fmt.Errorf("unable to add finalizer to landscape: %w", err)
		}
	}

	originalLandscape := landscape.DeepCopy()
	err := r.reconcileLandscape(ctx, landscape)

	err = pkgctrl.PatchStatusIfChanged(
		ctx,
		r.Client,
		landscape,
		originalLandscape,
		landscape.Status,
		originalLandscape.Status,
		"unable to update landscape status",
		err,
		"an error occurred while reconciling landscape",
	)
	if err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: defaultReconcileInterval}, nil
}

func (r *LandscapeReconciler) reconcileLandscape(ctx context.Context, landscape *konfidence.Landscape) error {
	log := logf.FromContext(ctx)
	log.Info("Reconciling landscape")

	// Retrieve project name
	projectName, err := r.getProjectName(ctx, landscape)
	if err != nil {
		msg := fmt.Sprintf("Failed to retrieve project name: %s", err.Error())
		r.setCondition(landscape, konfidence.LandscapeNamespaceReadyCondition, metav1.ConditionFalse,
			konfidence.LandscapeInvalidNamespaceReason, msg)
		r.setCondition(landscape, konfidence.LandscapeReadyCondition, metav1.ConditionFalse,
			konfidence.LandscapeInvalidNamespaceReason, msg)
		r.Recorder.Eventf(landscape, nil, corev1.EventTypeWarning,
			konfidence.LandscapeInvalidNamespaceReason, konfidence.LandscapeInvalidNamespaceReason, msg)
		// Do not return error - this is a permanent user error that won't fix itself.
		// Status and events are set, requeue at normal interval to avoid hot loop.
		return nil
	}

	landscape.Status.ProjectName = projectName

	ns, operationResult, err := r.createOrUpdateNamespace(ctx, landscape)
	if err != nil {
		reason := namespaceFailureReason(err)
		r.setCondition(landscape, konfidence.LandscapeNamespaceReadyCondition, metav1.ConditionFalse, reason, err.Error())
		r.setCondition(landscape, konfidence.LandscapeReadyCondition, metav1.ConditionFalse, reason, err.Error())
		r.Recorder.Eventf(landscape, nil, corev1.EventTypeWarning, reason, reason, err.Error())
		return err
	}

	landscape.Status.Namespace = ns.Name
	r.setCondition(landscape, konfidence.LandscapeNamespaceReadyCondition, metav1.ConditionTrue,
		konfidence.LandscapeNamespaceReconciledReason, fmt.Sprintf("Namespace %s reconciled", ns.Name))
	r.setCondition(landscape, konfidence.LandscapeReadyCondition, metav1.ConditionTrue,
		konfidence.LandscapeReconciledReason, fmt.Sprintf("Landscape %s reconciled", landscape.Name))

	if operationResult != controllerutil.OperationResultNone {
		reason := namespaceEventReason(operationResult)
		msg := fmt.Sprintf("Namespace %s %s for Landscape %s", ns.Name, operationResult, landscape.Name)
		r.Recorder.Eventf(landscape, nil, corev1.EventTypeNormal, reason, reason, msg)
		log.Info(msg)
	}
	return nil
}

// getProjectName retrieves the project name with priority:
// 1. status.projectName (if already reconciled)
// 2. parent namespace label (fresh lookup)
func (r *LandscapeReconciler) getProjectName(ctx context.Context, landscape *konfidence.Landscape) (string, error) {
	if landscape.Status.ProjectName != "" {
		return landscape.Status.ProjectName, nil
	}

	parentNS := &corev1.Namespace{}
	if err := r.Get(ctx, types.NamespacedName{Name: landscape.Namespace}, parentNS); err != nil {
		return "", fmt.Errorf("failed to get parent namespace %s: %w", landscape.Namespace, err)
	}

	projectName, hasProject := parentNS.Labels[pkgctrl.ProjectNameLabel]
	if !hasProject || projectName == "" {
		return "", fmt.Errorf("namespace %s is missing %s label", landscape.Namespace, pkgctrl.ProjectNameLabel)
	}

	return projectName, nil
}

func (r *LandscapeReconciler) createOrUpdateNamespace(
	ctx context.Context,
	landscape *konfidence.Landscape,
) (*corev1.Namespace, controllerutil.OperationResult, error) {
	projectName, err := r.getProjectName(ctx, landscape)
	if err != nil {
		return nil, controllerutil.OperationResultNone, fmt.Errorf("failed to get project name: %w", err)
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: landscapeNamespaceName(landscape, projectName),
		},
	}

	operationResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, ns, func() error {
		// Check if namespace already exists and is not managed by this landscape.
		// We cannot use metav1.IsControlledBy because Landscape is namespace-scoped
		// and cannot own a cluster-scoped Namespace. Instead, we rely on labels.
		if !ns.CreationTimestamp.IsZero() && !r.isManagedByLandscape(ns, landscape) {
			return fmt.Errorf("%w: namespace %s already exists and is not managed by Landscape %s/%s",
				errNamespaceConflict, ns.Name, landscape.Namespace, landscape.Name)
		}
		if !ns.DeletionTimestamp.IsZero() {
			return fmt.Errorf("%w: namespace %s is terminating", errNamespaceTerminating, ns.Name)
		}
		if ns.Labels == nil {
			ns.Labels = map[string]string{}
		}
		ns.Labels[pkgctrl.ManagedByLabel] = landscapeControllerName
		ns.Labels[pkgctrl.ProjectTypeLabel] = landscapeNamespaceTypeValue
		ns.Labels[pkgctrl.ProjectNameLabel] = projectName
		ns.Labels[pkgctrl.LandscapeNameLabel] = landscape.Name
		// Note: we cannot set an owner reference because Landscape is namespace-scoped
		// and Namespace is cluster-scoped. We rely on the finalizer for cleanup.
		return nil
	})
	if err != nil {
		return nil, operationResult, fmt.Errorf("failed to create or update namespace: %w", err)
	}
	return ns, operationResult, nil
}

// reconcileDelete deletes the managed landscape namespace (if any), waits for
// its termination and then releases the finalizer so the Landscape can be
// garbage-collected.
func (r *LandscapeReconciler) reconcileDelete(ctx context.Context, landscape *konfidence.Landscape) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(landscape, landscapeFinalizer) {
		return ctrl.Result{}, nil
	}

	ns := &corev1.Namespace{}
	err := r.Get(ctx, types.NamespacedName{Name: landscapeNamespaceName(landscape, landscape.Status.ProjectName)}, ns)
	if err != nil && !apierrors.IsNotFound(err) {
		return ctrl.Result{}, fmt.Errorf("unable to fetch landscape namespace: %w", err)
	}

	if err == nil && r.isManagedByLandscape(ns, landscape) {
		if ns.DeletionTimestamp.IsZero() {
			if delErr := r.Delete(ctx, ns); delErr != nil && !apierrors.IsNotFound(delErr) {
				return ctrl.Result{}, fmt.Errorf("unable to delete landscape namespace: %w", delErr)
			}
			msg := fmt.Sprintf("Namespace %s deleted for Landscape %s", ns.Name, landscape.Name)
			r.Recorder.Eventf(landscape, nil, corev1.EventTypeNormal, namespaceDeletedEventReason, namespaceDeletedEventReason, msg)
			log.Info(msg)
		}

		if err := r.setTerminatingStatus(ctx, landscape, ns.Name); err != nil {
			return ctrl.Result{}, err
		}
		log.Info("waiting for landscape namespace deletion", "namespace", ns.Name)
		return ctrl.Result{RequeueAfter: deletionRequeueInterval}, nil
	}

	controllerutil.RemoveFinalizer(landscape, landscapeFinalizer)
	if err := r.Update(ctx, landscape); err != nil {
		return ctrl.Result{}, fmt.Errorf("unable to remove finalizer from landscape: %w", err)
	}
	return ctrl.Result{}, nil
}

// setTerminatingStatus records, via a status patch, that the Landscape is waiting
// for its namespace to terminate, so a stuck deletion is visible in status.
func (r *LandscapeReconciler) setTerminatingStatus(ctx context.Context, landscape *konfidence.Landscape, nsName string) error {
	original := landscape.DeepCopy()
	msg := fmt.Sprintf("Waiting for namespace %s to terminate", nsName)
	r.setCondition(landscape, konfidence.LandscapeNamespaceReadyCondition, metav1.ConditionFalse,
		konfidence.LandscapeNamespaceTerminatingReason, msg)
	r.setCondition(landscape, konfidence.LandscapeReadyCondition, metav1.ConditionFalse,
		konfidence.LandscapeTerminatingReason, msg)

	return pkgctrl.PatchStatusIfChanged(
		ctx,
		r.Client,
		landscape,
		original,
		landscape.Status,
		original.Status,
		"unable to update landscape status",
		nil,
		"",
	)
}

// isManagedByLandscape checks if a namespace is managed by the given Landscape.
// Since Landscape is namespace-scoped and cannot set an owner reference on the
// cluster-scoped Namespace, we check labels instead.
func (r *LandscapeReconciler) isManagedByLandscape(ns *corev1.Namespace, landscape *konfidence.Landscape) bool {
	if ns.Labels == nil {
		return false
	}
	managedBy, hasManagedBy := ns.Labels[pkgctrl.ManagedByLabel]
	landscapeName, hasLandscape := ns.Labels[pkgctrl.LandscapeNameLabel]
	return hasManagedBy && managedBy == landscapeControllerName &&
		hasLandscape && landscapeName == landscape.Name
}

// isManagedByLandscapeController checks if a namespace is managed by the landscape
// controller (any landscape, not a specific one).
func isManagedByLandscapeController(ns *corev1.Namespace) bool {
	if ns.Labels == nil {
		return false
	}
	managedBy, hasManagedBy := ns.Labels[pkgctrl.ManagedByLabel]
	return hasManagedBy && managedBy == landscapeControllerName
}

func (r *LandscapeReconciler) setCondition(
	landscape *konfidence.Landscape,
	conditionType string,
	status metav1.ConditionStatus,
	reason, message string,
) {
	meta.SetStatusCondition(&landscape.Status.Conditions, metav1.Condition{
		Type:               conditionType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: landscape.Generation,
		LastTransitionTime: metav1.Now(),
	})
}

// landscapeNamespaceName returns the landscape namespace name using priority:
// 1. status.namespace (if already reconciled and stored)
// 2. spec.namespace (if user provided override)
// 3. computed default "kden-l-<landscape-name>-<hash>" (requires projectName)
// The hash is computed using FNV-1a 32-bit hash encoded in base36 for compactness.
func landscapeNamespaceName(landscape *konfidence.Landscape, projectName string) string {
	if landscape.Status.Namespace != "" {
		return landscape.Status.Namespace
	}

	if landscape.Spec.Namespace != "" {
		return landscape.Spec.Namespace
	}

	hashInput := projectName + ":" + landscape.Name
	hashStr := hash.Fnv(hashInput, 8)

	return fmt.Sprintf("%s%s-%s", konfidence.LandscapeNamespacePrefix, landscape.Name, hashStr)
}

// namespaceFailureReason maps a namespace reconciliation error to the
// matching condition and event reason.
func namespaceFailureReason(err error) string {
	switch {
	case errors.Is(err, errNamespaceConflict):
		return konfidence.LandscapeNamespaceConflictReason
	case errors.Is(err, errNamespaceTerminating):
		return konfidence.LandscapeNamespaceTerminatingReason
	default:
		return konfidence.LandscapeNamespaceCreateFailedReason
	}
}

func namespaceEventReason(operationResult controllerutil.OperationResult) string {
	if operationResult == controllerutil.OperationResultCreated {
		return namespaceCreatedEventReason
	}
	return namespaceUpdatedEventReason
}

// SetupWithManager sets up the controller with the Manager.
func (r *LandscapeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&konfidence.Landscape{}, builder.WithPredicates(reconcilePredicate())).
		Watches(
			// we cannot use .Owns(), because namespace-scoped Landscape cannot own a cluster-scoped Namespace
			&corev1.Namespace{},
			handler.EnqueueRequestsFromMapFunc(r.findLandscapeForNamespace),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Named(landscapeControllerName).
		Complete(r)
}

// findLandscapeForNamespace maps a Namespace event to reconcile requests for any
// Landscape resources that manage it. Since Landscape is namespace-scoped and
// cannot set an owner reference on the cluster-scoped Namespace, we identify
// the managing Landscape via labels.
func (r *LandscapeReconciler) findLandscapeForNamespace(ctx context.Context, obj client.Object) []reconcile.Request {
	ns, ok := obj.(*corev1.Namespace)
	if !ok {
		return nil
	}

	// Check if this namespace is managed by the landscape controller
	if !isManagedByLandscapeController(ns) {
		return nil
	}

	// Get the landscape name and parent project namespace from labels
	landscapeName, hasLandscape := ns.Labels[pkgctrl.LandscapeNameLabel]
	projectName, hasProject := ns.Labels[pkgctrl.ProjectNameLabel]
	if !hasLandscape || !hasProject {
		return nil
	}

	projectNsList := &corev1.NamespaceList{}
	if err := r.List(ctx, projectNsList, client.MatchingLabels{
		pkgctrl.ProjectTypeLabel: "project",
		pkgctrl.ProjectNameLabel: projectName,
	}); err != nil || len(projectNsList.Items) == 0 {
		return nil
	}

	return []reconcile.Request{
		{
			NamespacedName: types.NamespacedName{
				Name:      landscapeName,
				Namespace: projectNsList.Items[0].Name,
			},
		},
	}
}

// reconcilePredicate triggers on spec (generation) changes and on deletion.
// GenerationChangedPredicate alone would drop the deletion event (setting a
// deletionTimestamp does not bump generation), leaving the finalizer stuck.
func reconcilePredicate() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration() ||
				!e.ObjectNew.GetDeletionTimestamp().IsZero()
		},
	}
}

// NewLandscapeReconciler wires a LandscapeReconciler for the given manager.
func NewLandscapeReconciler(mgr ctrl.Manager) *LandscapeReconciler {
	return &LandscapeReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorder(landscapeControllerName),
	}
}

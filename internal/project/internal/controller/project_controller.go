package controller

import (
	"context"
	"errors"
	"fmt"
	"time"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	pkgctrl "github.com/konfidence-project/konfidence/pkg/controller"
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
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	defaultReconcileInterval = 30 * time.Second
	deletionRequeueInterval  = 5 * time.Second
	projectControllerName    = "project-controller"

	// adoptRequeueInterval bounds how long namespacePolicy Adopt waits before
	// re-checking for a target namespace that does not exist yet, rather than
	// relying on exponential error backoff.
	adoptRequeueInterval = 10 * time.Second

	// The project namespace must be fully deleted together with everything in
	// it before the Project is released. The finalizer lets the controller
	// delete the namespace and wait for its termination first.
	projectFinalizer = "konfidence.cloud/project-finalizer"

	// projectNamespaceTypeValue is the ProjectTypeLabel value on project namespaces.
	projectNamespaceTypeValue = "project"

	// Event reasons for namespace lifecycle transitions.
	namespaceCreatedEventReason  = "NamespaceCreated"
	namespaceUpdatedEventReason  = "NamespaceUpdated"
	namespaceDeletedEventReason  = "NamespaceDeleted"
	namespaceOrphanedEventReason = "NamespaceOrphaned"
)

var (
	errNamespaceConflict    = errors.New("namespace conflict")
	errNamespaceTerminating = errors.New("namespace terminating")
	errNamespaceNotFound    = errors.New("namespace not found")
)

// ProjectReconciler reconciles a Project object
type ProjectReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder events.EventRecorder
}

// +kubebuilder:rbac:groups=konfidence.cloud,resources=projects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=konfidence.cloud,resources=projects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=konfidence.cloud,resources=projects/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=events.k8s.io,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *ProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconcile project started...")

	project := &konfidence.Project{}
	if err := r.Get(ctx, req.NamespacedName, project); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !project.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, project)
	}

	if controllerutil.AddFinalizer(project, projectFinalizer) {
		if err := r.Update(ctx, project); err != nil {
			return ctrl.Result{}, fmt.Errorf("unable to add finalizer to project: %w", err)
		}
	}

	originalProject := project.DeepCopy()
	err := r.reconcileProject(ctx, project)

	err = pkgctrl.PatchStatusIfChanged(
		ctx,
		r.Client,
		project,
		originalProject,
		project.Status,
		originalProject.Status,
		"unable to update project status",
		err,
		"an error occurred while reconciling project",
	)
	if err != nil {
		// An Adopt waiting for its target namespace to appear is an expected,
		// self-resolving state rather than a failure: requeue on a fixed short
		// interval instead of letting the error drive exponential backoff.
		if errors.Is(err, errNamespaceNotFound) {
			return ctrl.Result{RequeueAfter: adoptRequeueInterval}, nil
		}
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: defaultReconcileInterval}, nil
}

func (r *ProjectReconciler) reconcileProject(ctx context.Context, project *konfidence.Project) error {
	log := logf.FromContext(ctx)
	log.Info("Reconciling project")

	ns, operationResult, err := r.createOrUpdateNamespace(ctx, project)
	if err != nil {
		reason := namespaceFailureReason(err)
		r.setCondition(project, konfidence.ProjectNamespaceReadyCondition, metav1.ConditionFalse, reason, err.Error())
		r.setCondition(project, konfidence.ProjectReadyCondition, metav1.ConditionFalse, reason, err.Error())
		r.Recorder.Eventf(project, nil, corev1.EventTypeWarning, reason, reason, err.Error())
		return err
	}

	project.Status.Namespace = ns.Name
	r.setCondition(project, konfidence.ProjectNamespaceReadyCondition, metav1.ConditionTrue,
		konfidence.ProjectNamespaceReconciledReason, fmt.Sprintf("Namespace %s reconciled", ns.Name))
	r.setCondition(project, konfidence.ProjectReadyCondition, metav1.ConditionTrue,
		konfidence.ProjectReconciledReason, fmt.Sprintf("Project %s reconciled", project.Name))

	// Only emit an event when the namespace actually changed
	if operationResult != controllerutil.OperationResultNone {
		reason := namespaceEventReason(operationResult)
		msg := fmt.Sprintf("Namespace %s %s for Project %s", ns.Name, operationResult, project.Name)
		r.Recorder.Eventf(project, nil, corev1.EventTypeNormal, reason, reason, msg)
		log.Info(msg)
	}
	return nil
}

// createOrUpdateNamespace writes the project namespace with managed-by labels
// so the controller owns its lifecycle and won't touch a foreign namespace.
func (r *ProjectReconciler) createOrUpdateNamespace(
	ctx context.Context,
	project *konfidence.Project,
) (*corev1.Namespace, controllerutil.OperationResult, error) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName(project),
		},
	}

	operationResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, ns, func() error {
		policy := project.Spec.NamespacePolicy
		if policy == "" {
			policy = konfidence.NamespacePolicyCreate
		}

		// A terminating namespace cannot be updated or recreated under any policy;
		// wait for the termination to complete and retry on a later reconciliation.
		if !ns.DeletionTimestamp.IsZero() {
			return fmt.Errorf("%w: namespace %s is terminating", errNamespaceTerminating, ns.Name)
		}

		// CreationTimestamp is zero only on the create path; a set value means the
		// namespace pre-exists. Adopt requires it to exist; Create makes it.
		if ns.CreationTimestamp.IsZero() {
			if policy == konfidence.NamespacePolicyAdopt {
				return fmt.Errorf("%w: namespace %s does not exist", errNamespaceNotFound, ns.Name)
			}
		} else if namespaceConflict(ns, project, policy) {
			return fmt.Errorf("%w: namespace %s already exists and is not managed by Project %s",
				errNamespaceConflict, ns.Name, project.Name)
		}

		if ns.Labels == nil {
			ns.Labels = map[string]string{}
		}
		ns.Labels[pkgctrl.ManagedByLabel] = projectControllerName
		ns.Labels[pkgctrl.ProjectTypeLabel] = projectNamespaceTypeValue
		ns.Labels[pkgctrl.ProjectNameLabel] = project.Name
		return nil
	})
	if err != nil {
		return nil, operationResult, fmt.Errorf("failed to create or update namespace: %w", err)
	}
	return ns, operationResult, nil
}

// reconcileDelete releases the project namespace according to the deletion
// policy and then removes the finalizer so the Project can be garbage-collected.
// Delete removes the namespace and waits for its termination; Orphan (the
// default) leaves the namespace in place and only strips its management labels.
func (r *ProjectReconciler) reconcileDelete(ctx context.Context, project *konfidence.Project) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(project, projectFinalizer) {
		return ctrl.Result{}, nil
	}

	ns := &corev1.Namespace{}
	err := r.Get(ctx, types.NamespacedName{Name: statusNamespaceName(project)}, ns)
	if err != nil && !apierrors.IsNotFound(err) {
		return ctrl.Result{}, fmt.Errorf("unable to fetch project namespace: %w", err)
	}

	if err == nil && isManagedBy(ns, project) {
		if project.Spec.DeletionPolicy == konfidence.DeletionPolicyDelete {
			// Delete the namespace and requeue until it is gone before releasing
			// the finalizer.
			if ns.DeletionTimestamp.IsZero() {
				if delErr := r.Delete(ctx, ns); delErr != nil && !apierrors.IsNotFound(delErr) {
					return ctrl.Result{}, fmt.Errorf("unable to delete project namespace: %w", delErr)
				}
				msg := fmt.Sprintf("Namespace %s deleted for Project %s", ns.Name, project.Name)
				r.Recorder.Eventf(project, nil, corev1.EventTypeNormal, namespaceDeletedEventReason, namespaceDeletedEventReason, msg)
				log.Info(msg)
			}
			// Surface the wait so a namespace whose termination is stuck (for example
			// on a resource with a blocking finalizer) is diagnosable from status
			// rather than a silent 5s requeue loop.
			if err := r.setTerminatingStatus(ctx, project, ns.Name); err != nil {
				return ctrl.Result{}, err
			}
			log.Info("waiting for project namespace deletion", "namespace", ns.Name)
			return ctrl.Result{RequeueAfter: deletionRequeueInterval}, nil
		}

		// Orphan: strip the management labels and leave the namespace behind.
		if err := r.orphanNamespace(ctx, project, ns); err != nil {
			return ctrl.Result{}, err
		}
	}

	controllerutil.RemoveFinalizer(project, projectFinalizer)
	if err := r.Update(ctx, project); err != nil {
		return ctrl.Result{}, fmt.Errorf("unable to remove finalizer from project: %w", err)
	}
	return ctrl.Result{}, nil
}

// orphanNamespace removes the konfidence.cloud/* management labels from the
// namespace so it is no longer owned by any Project, then emits an event.
func (r *ProjectReconciler) orphanNamespace(ctx context.Context, project *konfidence.Project, ns *corev1.Namespace) error {
	delete(ns.Labels, pkgctrl.ManagedByLabel)
	delete(ns.Labels, pkgctrl.ProjectTypeLabel)
	delete(ns.Labels, pkgctrl.ProjectNameLabel)
	if err := r.Update(ctx, ns); err != nil {
		return fmt.Errorf("unable to orphan project namespace: %w", err)
	}
	msg := fmt.Sprintf("Namespace %s orphaned for Project %s", ns.Name, project.Name)
	r.Recorder.Eventf(project, nil, corev1.EventTypeNormal, namespaceOrphanedEventReason, namespaceOrphanedEventReason, msg)
	logf.FromContext(ctx).Info(msg)
	return nil
}

// setTerminatingStatus records, via a status patch, that the Project is waiting
// for its namespace to terminate, so a stuck deletion is visible in status.
func (r *ProjectReconciler) setTerminatingStatus(ctx context.Context, project *konfidence.Project, nsName string) error {
	original := project.DeepCopy()
	msg := fmt.Sprintf("Waiting for namespace %s to terminate", nsName)
	r.setCondition(project, konfidence.ProjectNamespaceReadyCondition, metav1.ConditionFalse,
		konfidence.ProjectNamespaceTerminatingReason, msg)
	r.setCondition(project, konfidence.ProjectReadyCondition, metav1.ConditionFalse,
		konfidence.ProjectTerminatingReason, msg)

	return pkgctrl.PatchStatusIfChanged(
		ctx,
		r.Client,
		project,
		original,
		project.Status,
		original.Status,
		"unable to update project status",
		nil,
		"",
	)
}

func (r *ProjectReconciler) setCondition(
	project *konfidence.Project,
	conditionType string,
	status metav1.ConditionStatus,
	reason, message string,
) {
	meta.SetStatusCondition(&project.Status.Conditions, metav1.Condition{
		Type:               conditionType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: project.Generation,
		LastTransitionTime: metav1.Now(),
	})
}

// namespaceName returns the name of the project namespace: spec.namespace
// when set, "kden-project-<project-name>" otherwise.
func namespaceName(project *konfidence.Project) string {
	if project.Spec.Namespace != "" {
		return project.Spec.Namespace
	}
	return konfidence.ProjectNamespacePrefix + project.Name
}

// statusNamespaceName prefers the namespace recorded in the status and falls
// back to recomputing it, in case the Project is deleted before the status
// was ever written.
func statusNamespaceName(project *konfidence.Project) string {
	if project.Status.Namespace != "" {
		return project.Status.Namespace
	}
	return namespaceName(project)
}

// isManagedBy reports whether the namespace is owned by the given Project.
// Project is cluster-scoped but cannot own a cluster-scoped Namespace via a
// controller owner reference across the Project->Namespace boundary without
// tying namespace deletion to garbage collection, so ownership is tracked via
// the managed-by and project-name labels.
//
// Because ownership lives in these labels, manually editing managed-by or
// project-name on a managed namespace flips it to a conflict that the
// controller cannot repair on its own (unlike an owner reference). Recovering
// requires restoring the labels or deleting the namespace by hand.
func isManagedBy(ns *corev1.Namespace, project *konfidence.Project) bool {
	return ns.Labels[pkgctrl.ManagedByLabel] == projectControllerName &&
		ns.Labels[pkgctrl.ProjectNameLabel] == project.Name
}

// namespaceConflict reports whether an existing namespace may not be taken over
// by the given Project under the policy. Create refuses any namespace it does
// not already manage; Adopt refuses only one managed by a different konfidence
// resource, taking over an unmanaged namespace.
func namespaceConflict(ns *corev1.Namespace, project *konfidence.Project, policy konfidence.NamespacePolicy) bool {
	if isManagedBy(ns, project) {
		return false
	}
	if policy == konfidence.NamespacePolicyAdopt {
		return ns.Labels[pkgctrl.ManagedByLabel] != ""
	}
	return true
}

// namespaceFailureReason maps a namespace reconciliation error to the
// matching condition and event reason.
func namespaceFailureReason(err error) string {
	switch {
	case errors.Is(err, errNamespaceConflict):
		return konfidence.ProjectNamespaceConflictReason
	case errors.Is(err, errNamespaceTerminating):
		return konfidence.ProjectNamespaceTerminatingReason
	case errors.Is(err, errNamespaceNotFound):
		return konfidence.ProjectNamespaceNotFoundReason
	default:
		return konfidence.ProjectNamespaceCreateFailedReason
	}
}

func namespaceEventReason(operationResult controllerutil.OperationResult) string {
	if operationResult == controllerutil.OperationResultCreated {
		return namespaceCreatedEventReason
	}
	return namespaceUpdatedEventReason
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// We deliberately do not watch managed namespaces. Deletion or drift of the
	// management labels on a namespace should be very rare and does not require
	// immediate detection; the periodic requeue reconciles any drift as a backup.
	return ctrl.NewControllerManagedBy(mgr).
		For(&konfidence.Project{}, builder.WithPredicates(reconcilePredicate())).
		Named(projectControllerName).
		Complete(r)
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

// NewProjectReconciler wires a ProjectReconciler for the given manager.
func NewProjectReconciler(mgr ctrl.Manager) *ProjectReconciler {
	return &ProjectReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorder(projectControllerName),
	}
}

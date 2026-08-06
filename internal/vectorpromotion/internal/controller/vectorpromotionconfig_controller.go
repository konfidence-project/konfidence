package controller

import (
	"context"
	"errors"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const (
	VectorPromotionConfigControllerName = "vector-promotion-config-controller"

	EventActionDriftDetection = "DriftDetection"

	// Index keys for the reverse lookups behind the source watches. These are
	// manager-cache indexes only (RegisterFieldIndexes), not CRD
	// selectableFields: the map functions always run on the cached client.
	configSourceTemplateField = "config.sourceTemplate"
	configSourceStageField    = "config.sourceStage"
	configTargetStageField    = "config.targetStage"
	landscapeNamespaceField   = "landscape.statusNamespace"
)

// VectorPromotionConfigReconciler owns the config side of promotions: it keeps
// the Ready condition fresh by watching the referenced resources
// (vectorpromotionconfig_watches.go, vectorpromotionconfig_resolution.go),
// detects drift between the source vector and the target Stage, and creates
// sequence-stamped VectorPromotions (vectorpromotionconfig_drift.go) that the
// execution controller then runs.
type VectorPromotionConfigReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder events.EventRecorder
}

// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectorpromotionconfigs,verbs=get;list;watch
// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectorpromotionconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectorpromotions,verbs=get;list;watch;create
// +kubebuilder:rbac:groups=konfidence.cloud,resources=vectortemplates,verbs=get;list;watch
// +kubebuilder:rbac:groups=konfidence.cloud,resources=stages,verbs=get;list;watch
// +kubebuilder:rbac:groups=konfidence.cloud,resources=landscapes,verbs=get;list;watch

func (r *VectorPromotionConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	ctx = logf.IntoContext(ctx, log)

	config := &konfidence.VectorPromotionConfig{}
	if err := r.Get(ctx, req.NamespacedName, config); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	target, sourceVector, err := r.resolveReferences(ctx, config)
	var resErr *resolutionError
	if errors.As(err, &resErr) {
		return ctrl.Result{}, r.patchConfigStatus(ctx, config, func() {
			setConfigReadyCondition(config, metav1.ConditionFalse, resErr.reason, resErr.message)
		})
	}
	if err != nil {
		return ctrl.Result{}, err
	}

	if err := r.patchConfigStatus(ctx, config, func() {
		setConfigReadyCondition(config, metav1.ConditionTrue,
			konfidence.VectorPromotionConfigTargetResolvedReason, "source and target resolve")
	}); err != nil {
		return ctrl.Result{}, err
	}

	if sourceVector == "" || sourceVector == target.Spec.Vector {
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, r.createPromotionForDrift(ctx, config, sourceVector)
}

// NewVectorPromotionConfigReconciler wires a VectorPromotionConfigReconciler for the given manager.
func NewVectorPromotionConfigReconciler(mgr ctrl.Manager) *VectorPromotionConfigReconciler {
	return &VectorPromotionConfigReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorder(VectorPromotionConfigControllerName),
	}
}

// SetupWithManager sets up the controller with the Manager. Sources are
// watched cross-resource because the config cannot own them: ownership is
// reference-shaped and stages live in other namespaces, where owner
// references are not allowed.
func (r *VectorPromotionConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&konfidence.VectorPromotionConfig{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&konfidence.VectorPromotion{}).
		Watches(&konfidence.VectorTemplate{},
			handler.EnqueueRequestsFromMapFunc(r.mapVectorTemplateToConfigs),
			builder.WithPredicates(latestVectorChanged())).
		Watches(&konfidence.Stage{},
			handler.EnqueueRequestsFromMapFunc(r.mapStageToConfigs),
			builder.WithPredicates(stageVectorChanged())).
		Watches(&konfidence.Landscape{},
			handler.EnqueueRequestsFromMapFunc(r.mapLandscapeToConfigs),
			builder.WithPredicates(landscapeNamespaceChanged())).
		Named("vectorPromotionConfig").
		Complete(r)
}

package controller

import (
	"context"
	"errors"
	"fmt"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
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

	"github.com/konfidence-project/konfidence/internal/vectorpromotion/internal/promotion"
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
// the Ready condition fresh by watching the referenced resources, detects
// drift between the source vector and the target Stage, and creates
// sequence-stamped VectorPromotions that the execution controller then runs.
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

// resolveReferences resolves the target Stage and the source vector. A
// definitive miss is returned as a resolutionError for the Ready condition;
// an empty source vector with a nil error means the source exists but has not
// assembled a vector yet.
func (r *VectorPromotionConfigReconciler) resolveReferences(ctx context.Context, config *konfidence.VectorPromotionConfig) (*konfidence.Stage, string, error) {
	target, err := resolveTargetStage(ctx, r.Client, config)
	if err != nil {
		return nil, "", err
	}
	sourceVector, err := r.resolveSourceVector(ctx, config)
	if err != nil {
		return nil, "", err
	}
	return target, sourceVector, nil
}

// resolveSourceVector reads the vector currently offered by the source: a
// VectorTemplate's latest assembled vector, or the vector active on a source
// Stage.
func (r *VectorPromotionConfigReconciler) resolveSourceVector(ctx context.Context, config *konfidence.VectorPromotionConfig) (string, error) {
	source := config.Spec.Source
	if source.Kind == konfidence.VectorTemplateKind {
		template := &konfidence.VectorTemplate{}
		key := types.NamespacedName{Namespace: config.Namespace, Name: source.Name}
		err := r.Get(ctx, key, template)
		if apierrors.IsNotFound(err) {
			return "", &resolutionError{
				reason:  konfidence.VectorPromotionConfigSourceNotFoundReason,
				message: fmt.Sprintf("source vector template %q does not exist in namespace %q", key.Name, key.Namespace),
			}
		}
		if err != nil {
			return "", fmt.Errorf("failed to fetch source vector template %q: %w", key.Name, err)
		}
		return template.Status.LatestVector, nil
	}

	namespace, err := resolveLandscapeNamespace(ctx, r.Client, config.Namespace, source.Landscape)
	if err != nil {
		return "", err
	}
	stage := &konfidence.Stage{}
	key := types.NamespacedName{Namespace: namespace, Name: source.Name}
	err = r.Get(ctx, key, stage)
	if apierrors.IsNotFound(err) {
		return "", &resolutionError{
			reason:  konfidence.VectorPromotionConfigSourceNotFoundReason,
			message: fmt.Sprintf("source stage %q does not exist in landscape namespace %q", key.Name, key.Namespace),
		}
	}
	if err != nil {
		return "", fmt.Errorf("failed to fetch source stage %q in landscape namespace %q: %w", key.Name, key.Namespace, err)
	}
	return stage.Spec.Vector, nil
}

// createPromotionForDrift creates the next sequence-stamped promotion for the
// drifted source vector, unless a live promotion already pins it.
func (r *VectorPromotionConfigReconciler) createPromotionForDrift(ctx context.Context, config *konfidence.VectorPromotionConfig, sourceVector string) error {
	log := logf.FromContext(ctx)

	list := &konfidence.VectorPromotionList{}
	err := r.List(ctx, list,
		client.InNamespace(config.Namespace),
		client.MatchingFields{promotionConfigRefField: config.Name})
	if err != nil {
		return fmt.Errorf("failed to list promotions of config %q: %w", config.Name, err)
	}
	for i := range list.Items {
		if !promotion.IsTerminal(&list.Items[i]) && list.Items[i].Spec.Vector == sourceVector {
			return nil
		}
	}

	// The sequence is committed before the create: a crash in between leaves a
	// gap in the sequence, never a duplicate.
	if err := r.patchConfigStatus(ctx, config, func() { config.Status.Sequence++ }); err != nil {
		return err
	}

	vectorPromotion := &konfidence.VectorPromotion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      promotionName(config.Name, config.Status.Sequence),
			Namespace: config.Namespace,
		},
		Spec: konfidence.VectorPromotionSpec{
			VectorPromotionConfigRef: config.Name,
			Vector:                   sourceVector,
			RequireApproval:          config.Spec.Source.Kind == konfidence.StageKind,
			TTLAfterFinished:         config.Spec.TTLAfterFinished,
			Sequence:                 config.Status.Sequence,
		},
	}
	if err := controllerutil.SetControllerReference(config, vectorPromotion, r.Scheme); err != nil {
		return fmt.Errorf("failed to set owner reference on promotion %q: %w", vectorPromotion.Name, err)
	}
	if err := r.Create(ctx, vectorPromotion); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create promotion %q: %w", vectorPromotion.Name, err)
	}

	log.Info("created promotion for drifted source",
		"promotion", vectorPromotion.Name,
		"vector", sourceVector,
		"sequence", vectorPromotion.Spec.Sequence,
		"requireApproval", vectorPromotion.Spec.RequireApproval)
	r.Recorder.Eventf(config, vectorPromotion, corev1.EventTypeNormal, "VectorPromotionCreated",
		EventActionDriftDetection,
		fmt.Sprintf("created promotion %q for vector %q", vectorPromotion.Name, sourceVector))
	return nil
}

// patchConfigStatus applies mutate to the config status and patches it if it
// changed. Plain merge patch: this reconciler and the execution controller
// write disjoint config status fields (conditions/sequence here,
// lastPromotion* there), and controller-runtime serializes reconciles of the
// same config within this controller.
func (r *VectorPromotionConfigReconciler) patchConfigStatus(ctx context.Context, config *konfidence.VectorPromotionConfig, mutate func()) error {
	original := config.DeepCopy()
	mutate()
	if equalConfigStatus(config, original) {
		return nil
	}
	if err := r.Status().Patch(ctx, config, client.MergeFrom(original)); err != nil {
		return fmt.Errorf("failed to patch status of VectorPromotionConfig %q in namespace %q: %w",
			config.Name, config.Namespace, err)
	}
	return nil
}

// setConfigReadyCondition writes the config's Ready condition, telling users
// whether the resources their config references actually exist.
func setConfigReadyCondition(
	config *konfidence.VectorPromotionConfig,
	status metav1.ConditionStatus,
	reason, message string,
) {
	meta.SetStatusCondition(&config.Status.Conditions, metav1.Condition{
		Type:               konfidence.VectorPromotionConfigReadyCondition,
		Status:             status,
		ObservedGeneration: config.Generation,
		Reason:             reason,
		Message:            message,
	})
}

func equalConfigStatus(a, b *konfidence.VectorPromotionConfig) bool {
	if a.Status.Sequence != b.Status.Sequence {
		return false
	}
	current := meta.FindStatusCondition(a.Status.Conditions, konfidence.VectorPromotionConfigReadyCondition)
	previous := meta.FindStatusCondition(b.Status.Conditions, konfidence.VectorPromotionConfigReadyCondition)
	if current == nil || previous == nil {
		return current == previous
	}
	return current.Status == previous.Status && current.Reason == previous.Reason && current.Message == previous.Message
}

// promotionName builds `<config>-<sequence>`, trimming the config part when
// the result would exceed the DNS subdomain limit.
func promotionName(configName string, sequence int64) string {
	suffix := fmt.Sprintf("-%d", sequence)
	if len(configName)+len(suffix) > 253 {
		configName = configName[:253-len(suffix)]
	}
	return configName + suffix
}

// mapVectorTemplateToConfigs enqueues every config sourcing from the changed template.
func (r *VectorPromotionConfigReconciler) mapVectorTemplateToConfigs(ctx context.Context, obj client.Object) []reconcile.Request {
	return r.configRequests(ctx, obj.GetNamespace(), client.MatchingFields{configSourceTemplateField: obj.GetName()})
}

// mapStageToConfigs enqueues configs referencing the changed stage as source
// or target. The stage lives in a landscape namespace; its Landscape CR
// (indexed by status.namespace) points back to the project namespace where
// the configs live.
func (r *VectorPromotionConfigReconciler) mapStageToConfigs(ctx context.Context, obj client.Object) []reconcile.Request {
	log := logf.FromContext(ctx)

	landscapes := &konfidence.LandscapeList{}
	if err := r.List(ctx, landscapes, client.MatchingFields{landscapeNamespaceField: obj.GetNamespace()}); err != nil {
		log.Error(err, "failed to resolve landscape for stage namespace", "namespace", obj.GetNamespace())
		return nil
	}

	var requests []reconcile.Request
	for i := range landscapes.Items {
		landscape := &landscapes.Items[i]
		key := landscape.Name + "/" + obj.GetName()
		requests = append(requests,
			r.configRequests(ctx, landscape.Namespace, client.MatchingFields{configSourceStageField: key})...)
		requests = append(requests,
			r.configRequests(ctx, landscape.Namespace, client.MatchingFields{configTargetStageField: key})...)
	}
	return dedupeRequests(requests)
}

// mapLandscapeToConfigs enqueues configs referencing the changed landscape.
// Unindexed list plus filter: configs and the landscape share the project
// namespace, which holds few configs.
func (r *VectorPromotionConfigReconciler) mapLandscapeToConfigs(ctx context.Context, obj client.Object) []reconcile.Request {
	log := logf.FromContext(ctx)

	configs := &konfidence.VectorPromotionConfigList{}
	if err := r.List(ctx, configs, client.InNamespace(obj.GetNamespace())); err != nil {
		log.Error(err, "failed to list configs for landscape", "landscape", obj.GetName())
		return nil
	}
	var requests []reconcile.Request
	for i := range configs.Items {
		config := &configs.Items[i]
		if config.Spec.Source.Landscape == obj.GetName() || config.Spec.Target.Landscape == obj.GetName() {
			requests = append(requests, requestFor(config))
		}
	}
	return requests
}

func (r *VectorPromotionConfigReconciler) configRequests(ctx context.Context, namespace string, matching client.MatchingFields) []reconcile.Request {
	log := logf.FromContext(ctx)

	configs := &konfidence.VectorPromotionConfigList{}
	if err := r.List(ctx, configs, client.InNamespace(namespace), matching); err != nil {
		log.Error(err, "failed to list configs for watch mapping", "namespace", namespace)
		return nil
	}
	requests := make([]reconcile.Request, 0, len(configs.Items))
	for i := range configs.Items {
		requests = append(requests, requestFor(&configs.Items[i]))
	}
	return requests
}

func requestFor(config *konfidence.VectorPromotionConfig) reconcile.Request {
	return reconcile.Request{NamespacedName: types.NamespacedName{Namespace: config.Namespace, Name: config.Name}}
}

func dedupeRequests(requests []reconcile.Request) []reconcile.Request {
	seen := make(map[reconcile.Request]struct{}, len(requests))
	deduped := requests[:0]
	for _, request := range requests {
		if _, ok := seen[request]; ok {
			continue
		}
		seen[request] = struct{}{}
		deduped = append(deduped, request)
	}
	return deduped
}

func latestVectorChanged() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool { return true },
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldTemplate, okOld := e.ObjectOld.(*konfidence.VectorTemplate)
			newTemplate, okNew := e.ObjectNew.(*konfidence.VectorTemplate)
			return okOld && okNew && oldTemplate.Status.LatestVector != newTemplate.Status.LatestVector
		},
		DeleteFunc:  func(e event.DeleteEvent) bool { return true },
		GenericFunc: func(e event.GenericEvent) bool { return false },
	}
}

func stageVectorChanged() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool { return true },
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldStage, okOld := e.ObjectOld.(*konfidence.Stage)
			newStage, okNew := e.ObjectNew.(*konfidence.Stage)
			return okOld && okNew && oldStage.Spec.Vector != newStage.Spec.Vector
		},
		DeleteFunc:  func(e event.DeleteEvent) bool { return true },
		GenericFunc: func(e event.GenericEvent) bool { return false },
	}
}

func landscapeNamespaceChanged() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool { return true },
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldLandscape, okOld := e.ObjectOld.(*konfidence.Landscape)
			newLandscape, okNew := e.ObjectNew.(*konfidence.Landscape)
			return okOld && okNew && oldLandscape.Status.Namespace != newLandscape.Status.Namespace
		},
		DeleteFunc:  func(e event.DeleteEvent) bool { return true },
		GenericFunc: func(e event.GenericEvent) bool { return false },
	}
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

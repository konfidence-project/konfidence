package deploymenttarget

import (
	"context"
	"fmt"
	"strings"

	konfidencev1alpha1 "github.com/konfidence-project/konfidence/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/events"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlbuilder "sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const defaultControllerName = "deploymenttarget"

// ValidationResult describes whether DeploymentTarget parameters passed validation and provides a user-facing message.
type ValidationResult interface {
	// Valid reports whether the parameters passed validation.
	Valid() bool
	// Message returns additional information about the validation result.
	Message() string
}

type validationResult struct {
	success bool
	message string
}

func (v validationResult) Valid() bool {
	return v.success
}

func (v validationResult) Message() string {
	return v.message
}

// ValidationSuccess creates causes the controller to set the target's Ready condition to True with reason
// Accepted and the provided message.
func ValidationSuccess(messages ...string) ValidationResult {
	message := "parameters accepted"
	if len(messages) > 0 {
		message = messages[0]
	}

	return validationResult{
		success: true,
		message: message,
	}
}

// ValidationFailure creates causes the controller to set the target's Ready condition to False with
// reason InvalidParameters. It does not automatically retry reconciliation unless another event enqueues the
// target.
func ValidationFailure(message string) ValidationResult {
	return validationResult{
		success: false,
		message: message,
	}
}

// ValidationFunc validates spec.parameters, including the contents of referenced resources fetched with the
// client.Reader. A non-nil error means validation could not be completed; the result is ignored and reconciliation
// returns the error to trigger a retry. With a nil error, result must be non-nil and describes whether the parameters
// are valid. Use ValidationSuccess and ValidationFailure to report a ValidationResult.
type ValidationFunc func(ctx context.Context, c client.Reader, target *konfidencev1alpha1.DeploymentTarget) (ValidationResult, error)

type watch struct {
	object       client.Object
	eventHandler handler.TypedEventHandler[client.Object, reconcile.Request]
	opts         []ctrlbuilder.WatchesOption
}

type builder struct {
	manager                ctrl.Manager
	controllerName         string
	supportedArtifactTypes konfidencev1alpha1.ArtifactTypeSet
	validation             ValidationFunc
	watchers               []watch
}

// NewControllerManagedBy starts a builder bound to mgr.
func NewControllerManagedBy(mgr ctrl.Manager) *builder {
	return &builder{manager: mgr}
}

// For sets the artifact types this controller supports. It is required.
func (b *builder) For(supportedArtifactTypes konfidencev1alpha1.ArtifactTypeSet) *builder {
	b.supportedArtifactTypes = supportedArtifactTypes
	return b
}

// WithValidation sets the deployer's target-parameter validation function.
func (b *builder) WithValidation(validate ValidationFunc) *builder {
	b.validation = validate
	return b
}

// Watches adds a resource watch with the provided event handler.
func (b *builder) Watches(
	object client.Object,
	eventHandler handler.TypedEventHandler[client.Object, reconcile.Request],
	opts ...ctrlbuilder.WatchesOption,
) *builder {
	b.watchers = append(b.watchers, watch{
		object:       object,
		eventHandler: eventHandler,
		opts:         append([]ctrlbuilder.WatchesOption(nil), opts...),
	})
	return b
}

// Named sets the controller name. If omitted, it falls back to "deploymenttarget".
func (b *builder) Named(name string) *builder {
	b.controllerName = strings.TrimSpace(name)
	return b
}

// Complete validates the builder, constructs the reconciler, and registers it
// with the manager supplied to NewControllerManagedBy.
func (b *builder) Complete() error {
	if err := b.validateConfiguration(); err != nil {
		return err
	}
	if b.manager == nil {
		return fmt.Errorf("must provide a non-nil Manager to NewControllerManagedBy()")
	}

	if b.controllerName == "" {
		b.controllerName = defaultControllerName
	}
	r := b.build(b.manager.GetClient(), b.manager.GetEventRecorder(b.controllerName), b.controllerName)

	return b.complete(r)
}

func (b *builder) complete(r *reconciler) error {
	supportedTarget := predicate.NewPredicateFuncs(func(obj client.Object) bool {
		target, ok := obj.(*konfidencev1alpha1.DeploymentTarget)
		return ok && r.supportedArtifactTypes.Contains(target.ArtifactType())
	})

	cb := ctrlbuilder.TypedControllerManagedBy[reconcile.Request](b.manager).
		For(&konfidencev1alpha1.DeploymentTarget{}, ctrlbuilder.WithPredicates(predicate.GenerationChangedPredicate{}, supportedTarget))

	for _, watched := range b.watchers {
		eventHandler := watched.eventHandler
		if eventHandler == referencedTargetsMarker {
			gvk, err := watchedResourceGVK(watched.object, b.manager.GetScheme())
			if err != nil {
				return fmt.Errorf("resolve watched resource type: %w", err)
			}
			eventHandler = enqueueTargetsReferencing(gvk, r.supportedArtifactTypes, r.client)
		}
		cb = cb.Watches(watched.object, eventHandler, watched.opts...)
	}

	return cb.Named(r.controllerName).Complete(r)
}

func (b *builder) validateConfiguration() error {
	if len(b.supportedArtifactTypes) == 0 {
		return fmt.Errorf("must call For() with at least one supported artifact type")
	}
	for artifactType := range b.supportedArtifactTypes {
		if !artifactType.Valid() {
			return fmt.Errorf("call to For() included an invalid artifact type: %s", artifactType)
		}
	}
	for _, watcher := range b.watchers {
		if watcher.object == nil {
			return fmt.Errorf("calls to Watches() must provide a non-nil object")
		}
		if watcher.eventHandler == nil {
			return fmt.Errorf("calls to Watches() must provide a non-nil event handler")
		}
	}
	return nil
}

func watchedResourceGVK(object client.Object, scheme *runtime.Scheme) (schema.GroupVersionKind, error) {
	gvks, _, err := scheme.ObjectKinds(object)
	if err != nil {
		return schema.GroupVersionKind{}, err
	}
	if len(gvks) == 0 {
		return schema.GroupVersionKind{}, fmt.Errorf("type %T is not registered in the manager scheme", object)
	}
	gvk := gvks[0]
	for _, candidate := range gvks[1:] {
		if candidate.Group != gvk.Group || candidate.Kind != gvk.Kind {
			return schema.GroupVersionKind{}, fmt.Errorf("type %T maps to multiple API groups or kinds", object)
		}
	}
	return gvk, nil
}

func (b *builder) build(c client.Client, recorder events.EventRecorder, controllerName string) *reconciler {
	// copy relevant slices/maps to prevent modifications after initialization
	supported := make(konfidencev1alpha1.ArtifactTypeSet, len(b.supportedArtifactTypes))
	for artifactType := range b.supportedArtifactTypes {
		supported[artifactType] = struct{}{}
	}

	return &reconciler{
		client:                 c,
		controllerName:         controllerName,
		supportedArtifactTypes: supported,
		validateFunc:           b.validation,
		recorder:               recorder,
	}
}

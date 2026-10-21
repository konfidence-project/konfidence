package deploymenttarget

import (
	"context"
	"fmt"
	"strings"

	konfidencev1alpha1 "github.com/konfidence-project/konfidence/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlbuilder "sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const defaultControllerName = "deploymenttarget"

type ValidationFunc func(ctx context.Context, c client.Reader, target *konfidencev1alpha1.DeploymentTarget) error

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
	if b != nil {
		b.supportedArtifactTypes = supportedArtifactTypes
	}
	return b
}

// WithValidation sets the deployer's target-parameter validation function.
func (b *builder) WithValidation(validate ValidationFunc) *builder {
	if b != nil {
		b.validation = validate
	}
	return b
}

// Watches adds a resource watch with the provided event handler.
func (b *builder) Watches(
	object client.Object,
	eventHandler handler.TypedEventHandler[client.Object, reconcile.Request],
	opts ...ctrlbuilder.WatchesOption,
) *builder {
	if b != nil {
		b.watchers = append(b.watchers, watch{
			object:       object,
			eventHandler: eventHandler,
			opts:         append([]ctrlbuilder.WatchesOption(nil), opts...),
		})
	}
	return b
}

// Named sets the controller name. If omitted, it falls back to "deploymenttarget".
func (b *builder) Named(name string) *builder {
	if b != nil {
		b.controllerName = strings.TrimSpace(name)
	}
	return b
}

// Complete validates the builder, constructs the reconciler, and registers it
// with the manager supplied to NewControllerManagedBy.
func (b *builder) Complete() error {
	if err := b.validateConfiguration(); err != nil {
		return err
	}
	if b.manager == nil {
		return fmt.Errorf("must provide a non-nil Manager to NewDeploymentTargetControllerManagedBy()")
	}

	if b.controllerName == "" {
		b.controllerName = defaultControllerName
	}
	r, err := b.build(b.manager.GetClient(), b.controllerName)
	if err != nil {
		return err
	}

	return b.complete(r)
}

func (b *builder) complete(r *reconciler) error {
	supportedTarget := predicate.NewPredicateFuncs(func(obj client.Object) bool {
		target, ok := obj.(*konfidencev1alpha1.DeploymentTarget)
		return ok && r.supportedArtifactTypes.Contains(target.ArtifactType())
	})

	cb := ctrlbuilder.TypedControllerManagedBy[reconcile.Request](b.manager).
		For(&konfidencev1alpha1.DeploymentTarget{}, ctrlbuilder.WithPredicates(predicate.GenerationChangedPredicate{}, supportedTarget))

	for _, watched := range r.watchers {
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
	if b == nil {
		return fmt.Errorf("must provide a non-nil DeploymentTarget controller builder")
	}
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

func (b *builder) build(c client.Client, controllerName string) (*reconciler, error) {
	if c == nil {
		return nil, fmt.Errorf("must provide a non-nil Manager client")
	}

	// copy relevant slices/maps to prevent modifications after initialization
	supported := make(konfidencev1alpha1.ArtifactTypeSet, len(b.supportedArtifactTypes))
	for artifactType := range b.supportedArtifactTypes {
		supported[artifactType] = struct{}{}
	}
	watchers := append([]watch(nil), b.watchers...)
	for i := range watchers {
		watchers[i].opts = append([]ctrlbuilder.WatchesOption(nil), watchers[i].opts...)
	}

	return &reconciler{
		client:                 c,
		controllerName:         controllerName,
		supportedArtifactTypes: supported,
		watchers:               watchers,
		validateFunc:           b.validation,
	}, nil
}

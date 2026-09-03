package landscape

import (
	"context"
	"errors"
	"fmt"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ErrLandscapeNotFound is returned when a landscape filter names a landscape
// that does not exist in the project.
var ErrLandscapeNotFound = errors.New("landscape not found")

// ScopedLandscape is one landscape a project-scoped query may read from.
type ScopedLandscape struct {
	Landscape konfidence.Landscape

	// Namespace is the landscape's managed namespace. It is empty while the
	// landscape is still provisioning.
	Namespace string
}

type scopeOptions struct {
	// landscapeId is nil when no landscape filter was requested. An empty id is a
	// filter like any other - it just cannot match any landscape.
	landscapeId *string
}

// ScopeOption narrows the landscape scope resolved by ResolveScope.
type ScopeOption func(*scopeOptions)

// WithLandscapeId narrows the scope to the landscape with the given id. Passing an id
// no landscape can carry (in particular the empty id) narrows the scope to nothing and
// therefore yields ErrLandscapeNotFound.
func WithLandscapeId(id string) ScopeOption {
	return func(o *scopeOptions) {
		o.landscapeId = &id
	}
}

type Repository interface {
	Get(ctx context.Context, namespace, name string) (*konfidence.Landscape, error)
	ListForProject(ctx context.Context, namespace string) ([]konfidence.Landscape, error)

	// ResolveScope resolves the authoritative set of landscape namespaces a
	// project-scoped query may read from, optionally narrowed to a single
	// landscape. A landscape that is still provisioning stays in scope with an
	// empty Namespace. WithLandscapeId naming a landscape that does not exist
	// in the project returns ErrLandscapeNotFound.
	ResolveScope(ctx context.Context, projectNamespace string, opts ...ScopeOption) ([]ScopedLandscape, error)
}

type k8sRepository struct{ reader client.Reader }

func NewRepository(reader client.Reader) Repository {
	return &k8sRepository{reader: reader}
}

func (r *k8sRepository) Get(ctx context.Context, namespace, name string) (*konfidence.Landscape, error) {
	landscape := &konfidence.Landscape{}
	if err := r.reader.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, landscape); err != nil {
		return nil, fmt.Errorf("getting landscape %q in namespace %q: %w", name, namespace, err)
	}
	return landscape, nil
}

func (r *k8sRepository) ListForProject(ctx context.Context, namespace string) ([]konfidence.Landscape, error) {
	var list konfidence.LandscapeList
	if err := r.reader.List(ctx, &list, client.InNamespace(namespace)); err != nil {
		return nil, fmt.Errorf("listing landscapes in namespace %q: %w", namespace, err)
	}
	return list.Items, nil
}

func (r *k8sRepository) ResolveScope(
	ctx context.Context,
	projectNamespace string,
	opts ...ScopeOption,
) ([]ScopedLandscape, error) {
	options := &scopeOptions{}
	for _, opt := range opts {
		opt(options)
	}

	landscapes, err := r.ListForProject(ctx, projectNamespace)
	if err != nil {
		return nil, err
	}

	scope := make([]ScopedLandscape, 0, len(landscapes))
	for _, l := range landscapes {
		if options.landscapeId != nil && l.Name != *options.landscapeId {
			continue
		}
		scope = append(scope, ScopedLandscape{Landscape: l, Namespace: l.Status.Namespace})
	}

	if options.landscapeId != nil && len(scope) == 0 {
		return nil, fmt.Errorf("landscape %q in namespace %q: %w", *options.landscapeId, projectNamespace, ErrLandscapeNotFound)
	}

	return scope, nil
}

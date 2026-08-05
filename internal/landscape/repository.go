package landscape

import (
	"context"
	"fmt"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Repository interface {
	ListForProject(ctx context.Context, namespace string) ([]konfidence.Landscape, error)
}

type k8sRepository struct{ reader client.Reader }

func NewRepository(reader client.Reader) Repository {
	return &k8sRepository{reader: reader}
}

func (r *k8sRepository) ListForProject(ctx context.Context, namespace string) ([]konfidence.Landscape, error) {
	var list konfidence.LandscapeList
	if err := r.reader.List(ctx, &list, client.InNamespace(namespace)); err != nil {
		return nil, fmt.Errorf("listing landscapes in namespace %q: %w", namespace, err)
	}
	return list.Items, nil
}
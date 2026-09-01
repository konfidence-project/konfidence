package artifactdeployment

import (
	"context"
	"fmt"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	utils "github.com/konfidence-project/konfidence/pkg/controller"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ErrNotFound = fmt.Errorf("artifact deployment not found")

type Repository interface {
	Get(ctx context.Context, name string) (*konfidence.ArtifactDeployment, error)
	List(ctx context.Context, projectId string, opts ...ListOption) ([]konfidence.ArtifactDeployment, error)
}

type ListOption func(client.MatchingLabels)

func WithLandscapeId(id string) ListOption {
	return func(labels client.MatchingLabels) {
		labels[utils.LandscapeNameLabel] = id
	}
}

func WithVectorDeploymentId(id string) ListOption {
	return func(labels client.MatchingLabels) {
		labels[utils.VectorDeploymentNameLabel] = id
	}
}

type k8sRepository struct{ reader client.Reader }

func NewRepository(reader client.Reader) Repository {
	return &k8sRepository{reader: reader}
}

func (r *k8sRepository) Get(ctx context.Context, name string) (*konfidence.ArtifactDeployment, error) {
	var artifactDeployment konfidence.ArtifactDeployment
	if err := r.reader.Get(ctx, types.NamespacedName{Name: name}, &artifactDeployment); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("getting artifact deployment %q: %w", name, err)
	}

	return &artifactDeployment, nil
}

func (r *k8sRepository) List(ctx context.Context, projectId string, opts ...ListOption) ([]konfidence.ArtifactDeployment, error) {
	labelSelector := client.MatchingLabels{}

	for _, opt := range opts {
		opt(labelSelector)
	}

	var artifactDeployments konfidence.ArtifactDeploymentList
	listOpts := []client.ListOption{
		client.InNamespace(projectId),
		labelSelector,
	}

	if err := r.reader.List(ctx, &artifactDeployments, listOpts...); err != nil {
		return nil, fmt.Errorf("listing artifact deployments: %w", err)
	}

	return artifactDeployments.Items, nil
}

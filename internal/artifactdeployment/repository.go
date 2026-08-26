package artifactdeployment

import (
	"context"
	"fmt"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/auth"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ErrNotFound = fmt.Errorf("artifact deployment not found")
var ErrForbidden = fmt.Errorf("access to project not allowed")

type Repository interface {
	Get(ctx context.Context, name string, projectRoles auth.ProjectRoles) (*konfidence.ArtifactDeployment, error)
	List(ctx context.Context, projectId string, filters *ListFilters) ([]konfidence.ArtifactDeployment, error)
}

type ListFilters struct {
	LandscapeId        *string
	VectorDeploymentId *string
}

type k8sRepository struct{ reader client.Reader }

func NewRepository(reader client.Reader) Repository {
	return &k8sRepository{reader: reader}
}

func (r *k8sRepository) Get(ctx context.Context, name string, projectRoles auth.ProjectRoles) (*konfidence.ArtifactDeployment, error) {
	if len(projectRoles[name]) == 0 {
		return nil, ErrForbidden
	}

	var artifactDeployment konfidence.ArtifactDeployment
	if err := r.reader.Get(ctx, types.NamespacedName{Name: name}, &artifactDeployment); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("getting artifact deployment %q: %w", name, err)
	}
	return &artifactDeployment, nil
}

// TODO: these labels can be discussed in case there isnt a adr on how they are defined (atleast i didnt find one :D)
// this is in reference to criteria: add labels to CRDs for more efficient queries if necessary of the backlog
func (r *k8sRepository) List(ctx context.Context, namespace string, filters *ListFilters) ([]konfidence.ArtifactDeployment, error) {
	var artifactDeployments konfidence.ArtifactDeploymentList
	labelSelector := client.MatchingLabels{}

	if filters != nil {
		if filters.LandscapeId != nil {
			labelSelector["konfidence.cloud/landscape-name"] = *filters.LandscapeId
		}

		if filters.VectorDeploymentId != nil {
			labelSelector["konfidence.cloud/vector-deployment-name"] = *filters.VectorDeploymentId
		}
	}

	// Find by namespace + label selectors
	opts := []client.ListOption{
		client.InNamespace(namespace),
		labelSelector,
	}

	if err := r.reader.List(ctx, &artifactDeployments, opts...); err != nil {
		return nil, fmt.Errorf("listing artifact deployments: %w", err)
	}

	return artifactDeployments.Items, nil
}

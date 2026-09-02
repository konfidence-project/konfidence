package project

import (
	"context"
	"fmt"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/auth"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ErrNotFound = fmt.Errorf("project not found")

type Repository interface {
	// Get reads a project without checking authorization: callers authorize first
	// (see resolveProjectNamespace in the API handler).
	Get(ctx context.Context, name string) (*konfidence.Project, error)
	List(ctx context.Context, projectRoles auth.ProjectRoles) ([]konfidence.Project, error)
}

type k8sRepository struct{ reader client.Reader }

func NewRepository(reader client.Reader) Repository {
	return &k8sRepository{reader: reader}
}

func (r *k8sRepository) Get(ctx context.Context, name string) (*konfidence.Project, error) {
	var project konfidence.Project
	if err := r.reader.Get(ctx, types.NamespacedName{Name: name}, &project); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("getting project %q: %w", name, err)
	}

	return &project, nil
}

func (r *k8sRepository) List(ctx context.Context, projectRoles auth.ProjectRoles) ([]konfidence.Project, error) {
	var projects konfidence.ProjectList
	if err := r.reader.List(ctx, &projects); err != nil {
		return nil, fmt.Errorf("listing projects: %w", err)
	}

	authorizedProjectItems := make([]konfidence.Project, 0, len(projectRoles))
	for _, p := range projects.Items {
		if len(projectRoles[p.Name]) > 0 {
			authorizedProjectItems = append(authorizedProjectItems, p)
		}
	}

	return authorizedProjectItems, nil
}

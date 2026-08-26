package auth

import (
	"context"
	"fmt"
	"sort"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Repository that handle auth related functions
type Repository interface {
	GetProjectRoles(ctx context.Context, idpGroups []string) (ProjectRoles, error)
	GetAdminProjectRoles(ctx context.Context) (ProjectRoles, error)
}

type ProjectRoles map[string][]string

type k8sRepository struct{ reader client.Reader }

// NewRepository creates a Repository backed by the given reader. When reader is
// an informer cache, all Get and List calls are served from its local store.
func NewRepository(reader client.Reader) Repository {
	return &k8sRepository{reader: reader}
}

func (r *k8sRepository) GetProjectRoles(ctx context.Context, idpGroups []string) (ProjectRoles, error) {
	var projects konfidence.ProjectList

	if err := r.reader.List(ctx, &projects); err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	return mapProjectRoles(idpGroups, projects.Items), nil
}

func (r *k8sRepository) GetAdminProjectRoles(ctx context.Context) (ProjectRoles, error) {
	var projects konfidence.ProjectList
	if err := r.reader.List(ctx, &projects); err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	roles := make(ProjectRoles, len(projects.Items))
	for _, p := range projects.Items {
		roles[p.Name] = []string{"admin"}
	}
	return roles, nil
}

func mapProjectRoles(groups []string, projects []konfidence.Project) ProjectRoles {
	groupSet := make(map[string]struct{}, len(groups))
	for _, group := range groups {
		groupSet[group] = struct{}{}
	}

	projectRoles := make(ProjectRoles)
	for _, project := range projects {
		roles := matchingRoles(groupSet, project.Spec.RoleBindings)
		if len(roles) > 0 {
			projectRoles[project.Name] = roles
		}
	}
	return projectRoles
}

func matchingRoles(groups map[string]struct{}, roleBindings map[string]konfidence.Subjects) []string {
	roles := make([]string, 0, len(roleBindings))
	for role, subjects := range roleBindings {
		if matchesSessionGroup(groups, subjects) {
			roles = append(roles, role)
		}
	}
	sort.Strings(roles)
	return roles
}

func matchesSessionGroup(groups map[string]struct{}, subjects konfidence.Subjects) bool {
	for _, subject := range subjects {
		if subject.Session == nil {
			continue
		}
		for _, group := range subject.Session.MemberOf {
			if _, ok := groups[group]; ok {
				return true
			}
		}
	}
	return false
}

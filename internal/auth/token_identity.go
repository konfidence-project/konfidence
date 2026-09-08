package auth

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
)

type tokenBinding struct {
	project string
	role    string
	claims  map[string]konfidence.GlobMatch
}

func (r *k8sRepository) AuthenticateToken(ctx context.Context, rawToken string) (*TokenIdentity, error) {
	var projects konfidence.ProjectList
	if err := r.reader.List(ctx, &projects); err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	candidates := collectTokenBindings(projects.Items)
	var subject string
	verifiedAny := false
	projectRoles := make(map[string]map[string]struct{})
	for candidate, bindings := range candidates {
		token, err := r.tokenVerifier.Verify(ctx, rawToken, candidate.endpoint, candidate.audience)
		if err != nil {
			continue
		}

		if !verifiedAny {
			subject = token.subject
			verifiedAny = true
		}

		for _, binding := range bindings {
			if !matchesTokenClaims(token.claims, binding.claims) {
				continue
			}

			if projectRoles[binding.project] == nil {
				projectRoles[binding.project] = make(map[string]struct{})
			}
			projectRoles[binding.project][binding.role] = struct{}{}
		}
	}

	if !verifiedAny {
		return nil, ErrInvalidBearerToken
	}

	identity := &TokenIdentity{Subject: subject, ProjectRoles: make(ProjectRoles)}
	for project, roleSet := range projectRoles {
		roles := make([]string, 0, len(roleSet))
		for role := range roleSet {
			roles = append(roles, role)
		}
		sort.Strings(roles)
		identity.ProjectRoles[project] = roles
	}

	return identity, nil
}

func collectTokenBindings(projects []konfidence.Project) map[verifierKey][]tokenBinding {
	bindingsByCandidate := make(map[verifierKey][]tokenBinding)
	for _, project := range projects {
		for role, subjects := range project.Spec.RoleBindings {
			for _, subject := range subjects {
				if subject.JWKS == nil {
					continue
				}

				candidate := verifierKey{endpoint: subject.JWKS.Endpoint, audience: subject.JWKS.Audience}
				bindingsByCandidate[candidate] = append(
					bindingsByCandidate[candidate],
					tokenBinding{
						project: project.Name,
						role:    role,
						claims:  subject.JWKS.Claims,
					},
				)
			}
		}
	}

	return bindingsByCandidate
}

func matchesTokenClaims(claims map[string]any, expected map[string]konfidence.GlobMatch) bool {
	for name, pattern := range expected {
		value, ok := claims[name].(string)
		if !ok || !matchesGlob(string(pattern), value) {
			return false
		}
	}

	return true
}

func matchesGlob(pattern string, value string) bool {
	quoted := regexp.QuoteMeta(pattern)
	expression := "^" + strings.ReplaceAll(quoted, `\*`, ".*") + "$"
	matches, err := regexp.MatchString(expression, value)
	return err == nil && matches
}

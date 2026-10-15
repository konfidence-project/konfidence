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

type tokenBindingsByAudience map[string][]tokenBinding

func (r *k8sRepository) AuthenticateToken(ctx context.Context, rawToken string) (*TokenIdentity, error) {
	var projects konfidence.ProjectList
	if err := r.reader.List(ctx, &projects); err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	candidates := collectTokenBindings(projects.Items)
	var subject string
	verifiedAny := false
	projectRoles := make(map[string]map[string]struct{})
	for endpoint, bindingsByAudience := range candidates {
		allowedAudiences := make([]string, 0, len(bindingsByAudience))
		for audience := range bindingsByAudience {
			allowedAudiences = append(allowedAudiences, audience)
		}
		sort.Strings(allowedAudiences)

		token, err := r.tokenVerifier.Verify(ctx, rawToken, endpoint, allowedAudiences)
		if err != nil {
			continue
		}

		if !verifiedAny {
			subject = token.subject
			verifiedAny = true
		}

		// token.audiences contains only the intersection between the signed token's
		// audiences and the audiences configured for this endpoint.
		for _, audience := range token.audiences {
			for _, binding := range bindingsByAudience[audience] {
				if !matchesTokenClaims(token.claims, binding.claims) {
					continue
				}

				if projectRoles[binding.project] == nil {
					projectRoles[binding.project] = make(map[string]struct{})
				}
				projectRoles[binding.project][binding.role] = struct{}{}
			}
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

func collectTokenBindings(projects []konfidence.Project) map[string]tokenBindingsByAudience {
	bindingsByEndpoint := make(map[string]tokenBindingsByAudience)

	for _, project := range projects {
		for role, subjects := range project.Spec.RoleBindings {
			for _, subject := range subjects {
				if subject.JWKS == nil {
					continue
				}

				endpoint := subject.JWKS.Endpoint
				audience := subject.JWKS.Audience

				if bindingsByEndpoint[endpoint] == nil {
					bindingsByEndpoint[endpoint] = make(tokenBindingsByAudience)
				}

				bindingsByEndpoint[endpoint][audience] = append(
					bindingsByEndpoint[endpoint][audience],
					tokenBinding{
						project: project.Name,
						role:    role,
						claims:  subject.JWKS.Claims,
					},
				)
			}
		}
	}

	return bindingsByEndpoint
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

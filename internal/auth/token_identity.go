package auth

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
)

func (r *k8sRepository) AuthenticateToken(ctx context.Context, rawToken string) (*TokenIdentity, error) {
	var projects konfidence.ProjectList
	if err := r.reader.List(ctx, &projects); err != nil {
		return nil, fmt.Errorf(
			"failed to list projects: %w",
			err,
		)
	}

	candidates := collectVerifierCandidates(projects.Items)
	var subject string
	verified := make(map[verifierKey]*verifiedToken)
	for candidate := range candidates {
		token, err := r.tokenVerifier.Verify(ctx, rawToken, candidate.endpoint, candidate.audience)
		if err == nil {
			if len(verified) == 0 {
				subject = token.subject
			}
			verified[candidate] = token
		}
	}

	if len(verified) == 0 {
		return nil, ErrInvalidBearerToken
	}

	identity := &TokenIdentity{Subject: subject, ProjectRoles: make(ProjectRoles)}
	for _, project := range projects.Items {
		roles := matchingTokenRoles(project.Spec.RoleBindings, verified)
		if len(roles) > 0 {
			identity.ProjectRoles[project.Name] = roles
		}
	}

	return identity, nil
}

func collectVerifierCandidates(projects []konfidence.Project) map[verifierKey]struct{} {
	candidates := make(map[verifierKey]struct{})
	for _, project := range projects {
		for _, subjects := range project.Spec.RoleBindings {
			for _, subject := range subjects {
				if subject.JWKS == nil {
					continue
				}

				candidates[verifierKey{
					endpoint: subject.JWKS.Endpoint,
					audience: subject.JWKS.Audience,
				}] = struct{}{}
			}
		}
	}

	return candidates
}

func matchingTokenRoles(roleBindings map[string]konfidence.Subjects, verified map[verifierKey]*verifiedToken) []string {
	roles := make([]string, 0, len(roleBindings))
	for role, subjects := range roleBindings {
		if matchesTokenSubject(subjects, verified) {
			roles = append(roles, role)
		}
	}

	sort.Strings(roles)
	return roles
}

func matchesTokenSubject(subjects konfidence.Subjects, verified map[verifierKey]*verifiedToken) bool {
	for _, subject := range subjects {
		if subject.JWKS == nil {
			continue
		}

		token := verified[verifierKey{endpoint: subject.JWKS.Endpoint, audience: subject.JWKS.Audience}]
		if token == nil {
			continue
		}

		if matchesTokenClaims(token.claims, subject.JWKS.Claims) {
			return true
		}
	}

	return false
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

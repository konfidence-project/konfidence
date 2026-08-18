package cmd

import (
	"fmt"
	"path"
	"strings"
)

// FilterEnabledControllers parses a comma-separated list of [!]<glob> tokens and returns the
// subset of registered names that should be enabled, as a membership set.
//
// Semantics are set-based and order-independent. Tokens partition into
// positives and negatives; the result is (union of positive matches) minus
// (union of negative matches). A spec containing only negations subtracts
// from the full registered set, mirroring kube-controller-manager's
// --controllers flag, so "!Foo" means "everything except Foo". An empty spec
// or "*" yields all registered names. Glob matching uses path.Match, so "*",
// "?", and "[...]" are supported.
//
// Errors:
//   - a malformed glob (path.ErrBadPattern), and
//   - a literal token (no glob meta-characters) that matches zero registered
//     names — guards against typos and stale references to removed controllers.
func FilterEnabledControllers(spec string, registered []string) (map[string]bool, error) {
	enabled := make(map[string]bool, len(registered))

	if spec == "" || spec == "*" {
		for _, name := range registered {
			enabled[name] = true
		}
		return enabled, nil
	}

	positive := map[string]bool{}
	negative := map[string]bool{}
	hasPositive := false

	for raw := range strings.SplitSeq(spec, ",") {
		token := strings.TrimSpace(raw)
		if token == "" {
			continue
		}

		negate := false
		if strings.HasPrefix(token, "!") {
			negate = true
			token = strings.TrimSpace(token[1:])
			if token == "" {
				return nil, fmt.Errorf("invalid controller filter token: bare %q", "!")
			}
		} else {
			hasPositive = true
		}

		matched := false
		for _, name := range registered {
			ok, err := path.Match(token, name)
			if err != nil {
				return nil, fmt.Errorf("invalid controller filter glob %q: %w", token, err)
			}
			if !ok {
				continue
			}
			matched = true
			if negate {
				negative[name] = true
			} else {
				positive[name] = true
			}
		}

		if !matched {
			return nil, fmt.Errorf("controller filter token %q matches no registered controller", token)
		}
	}

	if !hasPositive {
		for _, name := range registered {
			positive[name] = true
		}
	}

	for name := range positive {
		if !negative[name] {
			enabled[name] = true
		}
	}
	return enabled, nil
}

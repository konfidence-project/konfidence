package cli

import (
	"fmt"
	"path"
	"strings"
)

// Filter parses a comma-separated list of [!]<glob> tokens and returns the
// subset of registered names that should be enabled, as a membership set.
//
// Semantics are set-based and order-independent. Tokens partition into
// positives and negatives; the result is (union of positive matches) minus
// (union of negative matches). An empty spec or "*" yields all registered
// names. Glob matching uses path.Match, so "*", "?", and "[...]" are
// supported.
//
// Errors:
//   - a malformed glob (path.ErrBadPattern), and
//   - a literal token (no glob meta-characters) that matches zero registered
//     names — guards against typos and stale references to removed controllers.
func Filter(spec string, registered []string) (map[string]bool, error) {
	enabled := make(map[string]bool, len(registered))

	if spec == "" || spec == "*" {
		for _, name := range registered {
			enabled[name] = true
		}
		return enabled, nil
	}

	positive := map[string]bool{}
	negative := map[string]bool{}

	for _, raw := range strings.Split(spec, ",") {
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

		if !matched && isLiteral(token) {
			return nil, fmt.Errorf("controller filter token %q matches no registered controller", token)
		}
	}

	for name := range positive {
		if !negative[name] {
			enabled[name] = true
		}
	}
	return enabled, nil
}

func isLiteral(token string) bool {
	return !strings.ContainsAny(token, "*?[")
}

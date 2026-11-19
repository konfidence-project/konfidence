package utils

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// NamespaceExclusions is an immutable startup policy for namespaced reconciliation.
// Its zero value excludes nothing. Cluster-scoped requests always pass through.
type NamespaceExclusions struct {
	namespaces map[string]struct{}
}

// ParseNamespaceExclusions parses the --exclude-namespaces comma-separated list.
func ParseNamespaceExclusions(spec string) (NamespaceExclusions, error) {
	exclusions := NamespaceExclusions{namespaces: make(map[string]struct{})}
	for _, value := range strings.Split(spec, ",") {
		namespace := strings.TrimSpace(value)
		if namespace == "" {
			continue
		}
		if problems := validation.IsDNS1123Label(namespace); len(problems) != 0 {
			return NamespaceExclusions{}, fmt.Errorf("invalid excluded namespace %q: %s", namespace, strings.Join(problems, "; "))
		}
		exclusions.namespaces[namespace] = struct{}{}
	}
	return exclusions, nil
}

// Contains reports whether a non-empty namespace is excluded.
func (e NamespaceExclusions) Contains(namespace string) bool {
	_, excluded := e.namespaces[namespace]
	return namespace != "" && excluded
}

// Wrap checks requests from every queue source before calling next.Reconcile.
func (e NamespaceExclusions) Wrap(next reconcile.Reconciler) reconcile.Reconciler {
	if len(e.namespaces) == 0 {
		return next
	}
	return reconcile.Func(func(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
		if e.Contains(req.Namespace) {
			return reconcile.Result{}, nil
		}
		return next.Reconcile(ctx, req)
	})
}

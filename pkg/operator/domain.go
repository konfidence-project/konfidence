// Package operator defines how controller domains plug into the operator
// binary. Each internal/<domain> package exports a Domain describing its
// --controllers flag name, the controllers it runs, and how to set them up;
// the binary only assembles the list.
package operator

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// Deps carries the runtime dependencies shared by every domain setup.
type Deps struct {
	Mgr    manager.Manager
	Logger logr.Logger

	// Limiter is the process-wide budget for all CPU-bound crypto work —
	// the verification matrix AND signing. For the global bound to hold, the
	// SAME instance must flow into both the shared Verifier's parallelism
	// (WithParallelism) and any signing domain's signer
	// (SignerBuilder.WithLimiter). Non-signing domains (e.g. vectordeployment)
	// do not need it — the Verifier they receive is already bounded. See
	// cmd/konfidence/cmd/operator.go for the construction site that
	// establishes this invariant.
	Limiter crypto.Limiter

	// Verifier is the process-wide, cache-backed OCM signature verifier
	// shared by every reconciler. It is stateless with respect to specs and
	// credentials; callers pass those per Verify call. Different CRs
	// verifying the same signature under the same spec share a single
	// verify + a single cache entry.
	Verifier crypto.Verifier

	// Shutdown stops the operator; long-running domain routines call it when
	// they fail terminally.
	Shutdown context.CancelFunc
}

// Domain is one --controllers toggle.
type Domain struct {
	// Name is the flag token that enables the domain.
	Name string

	// Controllers lists the controllers the domain runs, shown in the flag help.
	Controllers string

	Setup func(ctx context.Context, deps Deps) error
}

// Names returns the flag names of the given domains.
func Names(domains []Domain) []string {
	names := make([]string, len(domains))
	for i, domain := range domains {
		names[i] = domain.Name
	}
	return names
}

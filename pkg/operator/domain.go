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

	// Limiter bounds process-wide CPU-bound crypto work across all domains.
	Limiter crypto.Limiter

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

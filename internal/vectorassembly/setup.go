package vectorassembly

import (
	"context"
	"fmt"

	lru "github.com/hashicorp/golang-lru/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/konfidence-project/konfidence/internal/vectorassembly/internal/controller"
	"github.com/konfidence-project/konfidence/internal/vectorassembly/internal/vector"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	"github.com/konfidence-project/konfidence/pkg/operator"
)

const OperatorFlagName = "VectorAssembly"

// Domain wires the vector assembly controllers into the operator's --controllers flag.
func Domain() operator.Domain {
	return operator.Domain{
		Name:        OperatorFlagName,
		Controllers: "VectorTemplate",
		Setup: func(_ context.Context, deps operator.Deps) error {
			return SetupControllers(deps.Mgr, Options{
				Limiter:  deps.Limiter,
				Verifier: deps.Verifier,
			})
		},
	}
}

// Options configures the vector assembly controllers.
type Options struct {
	// Limiter bounds process-wide CPU-bound crypto work. Required; use crypto.NewLimiter(0) for GOMAXPROCS.
	Limiter crypto.Limiter

	// Verifier is the shared process-wide OCM verifier. Required; usually from operator.Deps.Verifier.
	Verifier crypto.Verifier
}

// SetupControllers registers all vector assembly controllers with the given manager.
func SetupControllers(mgr ctrl.Manager, opts Options) error {
	if opts.Limiter == nil {
		return fmt.Errorf("setup: Limiter is required; use crypto.NewLimiter(0) for GOMAXPROCS")
	}
	if opts.Verifier == nil {
		return fmt.Errorf("setup: Verifier is required; usually from operator.Deps.Verifier")
	}

	log := logf.Log.WithName("vectorassembly")

	vectorCache, err := lru.New[string, vector.Vector](controller.VectorCacheSize)
	if err != nil {
		return fmt.Errorf("creating vector cache: %w", err)
	}

	if err := controller.NewVectorTemplateReconciler(
		mgr,
		opts.Verifier,
		opts.Limiter,
		log,
		vectorCache,
		vector.TimestampVectorVersionGenerator,
	).SetupWithManager(mgr); err != nil {
		return err
	}

	return nil
}

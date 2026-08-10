package vectorassembly

import (
	"context"
	"fmt"

	lru "github.com/hashicorp/golang-lru/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"github.com/konfidence-project/konfidence/internal/vectorassembly/internal/controller"
	"github.com/konfidence-project/konfidence/internal/vectorassembly/internal/vector"
	"github.com/konfidence-project/konfidence/pkg/lrucache"
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
			return SetupControllers(deps.Mgr, Options{Limiter: deps.Limiter})
		},
	}
}

// Options configures the vector assembly controllers.
type Options struct {
	// Limiter bounds process-wide CPU-bound crypto work. Required; use crypto.NewLimiter(0) for GOMAXPROCS.
	Limiter crypto.Limiter
}

// SetupControllers registers all vector assembly controllers with the given manager.
func SetupControllers(mgr ctrl.Manager, opts Options) error {
	if opts.Limiter == nil {
		return fmt.Errorf("setup: Limiter is required; use crypto.NewLimiter(0) for GOMAXPROCS")
	}

	log := logf.Log.WithName("vectorassembly")

	cache, err := lrucache.New(
		lrucache.DefaultCacheSize,
		lrucache.CRExtract[*konfidence.VectorTemplate],
		controller.NewCacheFactory(log, opts.Limiter),
	)
	if err != nil {
		return fmt.Errorf("creating cache: %w", err)
	}

	vectorCache, err := lru.New[string, vector.Vector](controller.VectorCacheSize)
	if err != nil {
		return fmt.Errorf("creating vector cache: %w", err)
	}

	if err := controller.NewVectorTemplateReconciler(mgr, cache, vectorCache, vector.TimestampVectorVersionGenerator).
		SetupWithManager(mgr); err != nil {
		return err
	}

	return nil
}

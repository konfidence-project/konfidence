package controller

import (
	"context"

	pkgrepository "github.com/konfidence-project/konfidence/pkg/ocm/repository"
	"ocm.software/open-component-model/bindings/go/oci/compref"
)

// OcmPort is an interface that abstracts the promotion process
// from a source to a target vector using OCM.
type OcmPort interface {
	Promote(ctx context.Context, source, target compref.Ref) error
}

// OcmPortProvider is a factory interface for creating OcmPort instances.
type OcmPortProvider interface {
	NewOcmPromotionPort(client pkgrepository.Client) OcmPort
}

// OcmPortProviderFunc is a function bridge type that implements OcmPortProvider.
type OcmPortProviderFunc func(client pkgrepository.Client) OcmPort

func (f OcmPortProviderFunc) NewOcmPromotionPort(client pkgrepository.Client) OcmPort {
	return f(client)
}

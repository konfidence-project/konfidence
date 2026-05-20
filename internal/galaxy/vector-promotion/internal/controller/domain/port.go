package domain

import (
	"context"

	pkgrepository "github.com/konfidence-project/konfidence/pkg/ocm/repository"
	"ocm.software/open-component-model/bindings/go/oci/compref"
)

// OcmPromotionPort is an interface that abstracts the promotion process
// from a source to a target vector using OCM.
type OcmPromotionPort interface {
	Promote(ctx context.Context, source, target compref.Ref) error
}

// OcmPromotionPortProvider is a factory interface for creating OcmPromotionPort instances.
type OcmPromotionPortProvider interface {
	NewOcmPromotionPort(client pkgrepository.Client) OcmPromotionPort
}

// OcmPromotionPortProviderFunc is a function bridge type that implements OcmPromotionPortProvider.
type OcmPromotionPortProviderFunc func(client pkgrepository.Client) OcmPromotionPort

func (f OcmPromotionPortProviderFunc) NewOcmPromotionPort(client pkgrepository.Client) OcmPromotionPort {
	return f(client)
}

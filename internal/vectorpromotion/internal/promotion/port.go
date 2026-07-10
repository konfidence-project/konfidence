package promotion

import (
	"context"

	"ocm.software/open-component-model/bindings/go/oci/compref"
)

// OcmPort is an interface that abstracts the promotion process
// from a source to a target vector using OCM.
type OcmPort interface {
	Promote(ctx context.Context, source, target compref.Ref) error
}

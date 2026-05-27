package ocm

//go:generate go run go.uber.org/mock/mockgen -destination=mock/client_mock.go -package=mock github.com/konfidence-project/konfidence/pkg/ocm/repository Client
//go:generate go run go.uber.org/mock/mockgen -destination=mock/verifier_mock.go -package=mock github.com/konfidence-project/konfidence/pkg/ocm/crypto Verifier

import (
	"context"
	"errors"
	"fmt"

	"github.com/konfidence-project/konfidence/internal/galaxy/vectorpromotion/internal/promotion"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	pkgrepository "github.com/konfidence-project/konfidence/pkg/ocm/repository"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	ocispec "ocm.software/open-component-model/bindings/go/oci/spec/repository/v1/oci"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	_ promotion.OcmPort = (*PromotionAdapter)(nil)
)

// PromotionAdapter implements promotion.OcmPort using pkgrepository.Client.
type PromotionAdapter struct {
	ocmClient      pkgrepository.Client
	vectorVerifier crypto.Verifier
}

// NewPromotionAdapter creates a new PromotionAdapter with the given options.
func NewPromotionAdapter(opts ...PromotionAdapterOption) *PromotionAdapter {
	adapter := &PromotionAdapter{}
	for _, opt := range opts {
		opt(adapter)
	}
	if adapter.vectorVerifier == nil {
		ctrl.Log.Info("vector verifier not configured - using noop verifier")
		adapter.vectorVerifier = crypto.NoopVerifier{}
	}
	return adapter
}

// NewPromotionPortProvider creates a PromotionPortProvider that builds a PromotionAdapter
// with the given options and plugs in the provided client at call time.
func NewPromotionPortProvider(opts ...PromotionAdapterOption) promotion.OcmPortProviderFunc {
	return func(client pkgrepository.Client) promotion.OcmPort {
		a := NewPromotionAdapter(opts...)
		a.ocmClient = client
		return a
	}
}

func (a *PromotionAdapter) Promote(ctx context.Context, source, target compref.Ref) error {
	sourceDesc, err := a.ocmClient.Get(ctx, source)
	if err != nil {
		return fmt.Errorf("failed to get descriptor of source reference: %w", errors.Join(promotion.ErrFetchingSourceFailed, err))
	}
	if err = a.vectorVerifier.Verify(ctx, &sourceDesc); err != nil {
		return fmt.Errorf("unable to verify signature of the source reference descriptor: %w", errors.Join(promotion.ErrSourceVerificationFailed, err))
	}

	sourceVersionRef := compref.Ref{
		Repository: source.Repository,
		Component:  sourceDesc.Component.Name,
		Version:    sourceDesc.Component.Version,
	}

	same, err := sameLocation(source, target)
	if err != nil {
		return fmt.Errorf("failed to process promotion: %w", err)
	}
	if !same {
		if err := a.ocmClient.Copy(ctx, []compref.Ref{sourceVersionRef}, target.Repository); err != nil { // TODO: check if alias tag gets copied or only semver tag
			return fmt.Errorf("failed to copy %s -> %s during promotion: %w", source, target, err)
		}
	}

	aliasRef := compref.Ref{
		Repository: target.Repository,
		Component:  sourceVersionRef.Component,
		Version:    sourceVersionRef.Version,
	}
	if err := a.ocmClient.AddAlias(ctx, aliasRef, target.Version); err != nil {
		return fmt.Errorf("failed to add alias %q to %s during promotion: %w", target.Version, aliasRef, err)
	}

	return nil
}

// sameLocation returns true if two refs point to the same location (same base URL and subpath)
// it returns an error in case a ref's repository does not point to a ocispec.Repository.
func sameLocation(a, b compref.Ref) (bool, error) {
	aOCI, ok := a.Repository.(*ocispec.Repository)
	if !ok {
		return false, fmt.Errorf("source repository is not an OCI repository: %s", a)
	}
	bOCI, ok := b.Repository.(*ocispec.Repository)
	if !ok {
		return false, fmt.Errorf("target repository is not an OCI repository: %s", b)
	}
	return aOCI.BaseUrl == bOCI.BaseUrl && aOCI.SubPath == bOCI.SubPath, nil
}

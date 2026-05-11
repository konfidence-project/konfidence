/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ocm

//go:generate go run go.uber.org/mock/mockgen -destination=internal/mock/client_mock.go -package=mock github.com/konfidence-project/pkg/ocm/repository Client
//go:generate go run go.uber.org/mock/mockgen -destination=internal/mock/verifier_mock.go -package=mock github.com/konfidence-project/pkg/ocm/crypto Verifier

import (
	"context"
	"errors"
	"fmt"

	"github.com/konfidence-project/gcp-vector-promotion-controller/internal/controller/domain"
	"github.com/konfidence-project/pkg/ocm/crypto"
	pkgrepository "github.com/konfidence-project/pkg/ocm/repository"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	ocispec "ocm.software/open-component-model/bindings/go/oci/spec/repository/v1/oci"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	_ domain.OcmPromotionPort = (*PromotionAdapter)(nil)
)

// PromotionAdapter implements domain.OcmPromotionPort using pkgrepository.Client.
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
func NewPromotionPortProvider(opts ...PromotionAdapterOption) domain.OcmPromotionPortProviderFunc {
	return func(client pkgrepository.Client) domain.OcmPromotionPort {
		a := NewPromotionAdapter(opts...)
		a.ocmClient = client
		return a
	}
}

func (a *PromotionAdapter) Promote(ctx context.Context, source, target compref.Ref) error {
	sourceDesc, err := a.ocmClient.Get(ctx, source)
	if err != nil {
		return fmt.Errorf("failed to get descriptor of source reference: %w", errors.Join(domain.ErrFetchingSourceFailed, err))
	}
	if err = a.vectorVerifier.Verify(ctx, &sourceDesc); err != nil {
		return fmt.Errorf("unable to verify signature of the source reference descriptor: %w", errors.Join(domain.ErrSourceVerificationFailed, err))
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

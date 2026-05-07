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

import (
	"context"
	"errors"
	"fmt"

	"github.com/konfidence-project/gcp-vector-promotion-controller/internal/controller/domain"
	konfcompref "github.com/konfidence-project/pkg/ocm/compref"
	pkgrepository "github.com/konfidence-project/pkg/ocm/repository"
	"ocm.software/open-component-model/bindings/go/oci/compref"
	ocispec "ocm.software/open-component-model/bindings/go/oci/spec/repository/v1/oci"
)

var _ domain.PromotionPort = (*PromotionAdapter)(nil)

// PromotionAdapter implements PromotionPort using the pkg module's OCM client.
type PromotionAdapter struct {
	ocmClient pkgrepository.Client
}

// NewPromotionAdapter creates a new PromotionAdapter with the given OCM client.
func NewPromotionAdapter(ocmClient pkgrepository.Client) *PromotionAdapter {
	return &PromotionAdapter{ocmClient: ocmClient}
}

func (a *PromotionAdapter) Promote(ctx context.Context, source, target string) error {
	sourceRef, err := konfcompref.Parse(source)
	if err != nil {
		return fmt.Errorf("failed to parse source reference %q: %w", source, errors.Join(domain.ErrInvalidConfiguration, err))
	}

	targetRef, err := konfcompref.Parse(target, konfcompref.WithVersionValidation(konfcompref.VersionValidationAliasOnly))
	if err != nil {
		return fmt.Errorf("failed to parse target reference %q: %w", target, errors.Join(domain.ErrInvalidConfiguration, err))
	}

	sourceDesc, err := a.ocmClient.Get(ctx, *sourceRef)
	if err != nil {
		return fmt.Errorf("failed to get source reference: %w", errors.Join(domain.ErrFetchingSourceFailed, err))
	}
	sourceVersionRef := compref.Ref{
		Repository: sourceRef.Repository,
		Component:  sourceDesc.Component.Name,
		Version:    sourceDesc.Component.Version,
	}

	same, err := sameLocation(sourceRef, targetRef)
	if err != nil {
		return fmt.Errorf("failed to process promotion: %w", err)
	}
	if !same {
		if err := a.ocmClient.Copy(ctx, []compref.Ref{sourceVersionRef}, targetRef.Repository); err != nil { // TODO: check if alias tag gets copied or only semver tag
			return fmt.Errorf("failed to copy %s -> %s during promotion: %w", sourceRef, targetRef, err)
		}
	}

	aliasRef := compref.Ref{
		Repository: targetRef.Repository,
		Component:  sourceVersionRef.Component,
		Version:    sourceVersionRef.Version,
	}
	// Move the target alias to point to the resolved version.
	if err := a.ocmClient.AddAlias(ctx, aliasRef, targetRef.Version); err != nil {
		return fmt.Errorf("failed to add alias %q to %s during promotion: %w", targetRef.Version, aliasRef, err)
	}

	return nil
}

// sameLocation returns true if two refs point to the same location (same base URL and subpath)
// it returns an error in case a ref's repository does not point to a ocispec.Repository.
func sameLocation(a, b *compref.Ref) (bool, error) {
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

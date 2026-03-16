package ocm

import (
	"context"
	"fmt"

	"github.com/konfidence-project/pkg/ocm/crypto"
	pkgOcm "github.com/konfidence-project/pkg/ocm/repository"
	"ocm.software/open-component-model/bindings/go/oci/compref"
)

var ErrVersionInReference = fmt.Errorf("version in vector component reference is not allowed in StageConfiguration")

type VectorOCMAdapter struct {
	VectorVerifier crypto.Verifier
	OcmClient      pkgOcm.Client
}

func (o VectorOCMAdapter) GetLatestVectorVersion(ctx context.Context, registryAndComponent string) (string, error) {
	vectorOCMComponent, err := parseAndValidateComponentReference(registryAndComponent)
	if err != nil {
		return "", err
	}

	descriptor, err := o.OcmClient.Get(ctx, *vectorOCMComponent)
	if err != nil {
		return "", fmt.Errorf("unable to get latest version for vector ocm component (%s): %w", registryAndComponent, err)
	}

	version := descriptor.Component.Version
	if err := o.VectorVerifier.Verify(ctx, &descriptor); err != nil {
		return "", fmt.Errorf("unable to verify ocm descriptor for vector (%s) version (%s): %w",
			registryAndComponent, version, err)
	}

	return version, nil
}

func parseAndValidateComponentReference(reference string) (*compref.Ref, error) {
	parsedRef, err := compref.Parse(reference)
	if err != nil {
		return nil, err
	}
	if parsedRef.Version != "" {
		return nil, ErrVersionInReference
	}
	return parsedRef, nil
}

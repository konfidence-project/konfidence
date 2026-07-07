package ocm

import (
	"context"
	"fmt"

	"github.com/konfidence-project/konfidence/internal/galaxy/stageconfiguration/internal/ports"
	pkgcomp "github.com/konfidence-project/konfidence/pkg/ocm/compref"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	pkgrepo "github.com/konfidence-project/konfidence/pkg/ocm/repository"
)

var _ ports.VectorPort = VectorOCMAdapter{}

type VectorOCMAdapter struct {
	VectorVerifier crypto.Verifier
	OcmClient      pkgrepo.Client
}

func (o VectorOCMAdapter) GetLatestVectorVersion(ctx context.Context, registryAndComponent string) (string, error) {
	vectorOCMComponent, err := pkgcomp.Parse(registryAndComponent)
	if err != nil {
		return "", err
	}

	descriptor, err := o.OcmClient.Get(ctx, *vectorOCMComponent)
	if err != nil {
		return "", fmt.Errorf("unable to get specified version for vector ocm component (%s): %w", registryAndComponent, err)
	}

	version := descriptor.Component.Version
	if err := o.VectorVerifier.Verify(ctx, &descriptor); err != nil {
		return "", fmt.Errorf("unable to verify ocm descriptor for vector (%s) version (%s): %w",
			registryAndComponent, version, err)
	}

	vectorOCMComponent.Type = ""
	vectorOCMComponent.Version = version
	return vectorOCMComponent.String(), nil
}

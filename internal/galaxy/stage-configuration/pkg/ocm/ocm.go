package ocm

import (
	"context"
	"fmt"

	pkgComp "github.com/konfidence-project/pkg/ocm/compref"
	"github.com/konfidence-project/pkg/ocm/crypto"

	"github.com/konfidence-project/gcp-stage-configuration-controller/internal/controller/ports"
	pkgRepo "github.com/konfidence-project/pkg/ocm/repository"
)

type VectorOCMAdapter struct {
	VectorVerifier crypto.Verifier
	OcmClient      pkgRepo.Client
}

func (o VectorOCMAdapter) GetLatestVectorVersion(ctx context.Context, registryAndComponent string) (string, error) {
	vectorOCMComponent, err := pkgComp.Parse(registryAndComponent)
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

var DefaultPortProvider = ports.VectorPortProviderFunc(func(verifier crypto.Verifier, client pkgRepo.Client) ports.VectorPort {
	return VectorOCMAdapter{
		VectorVerifier: verifier,
		OcmClient:      client,
	}
})

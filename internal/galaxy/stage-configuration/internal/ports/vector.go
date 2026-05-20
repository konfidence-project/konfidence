package ports

//go:generate go run go.uber.org/mock/mockgen -source=vector.go -destination=mocks/mock_vector_port.go -package=mocks

import (
	"context"

	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	pkgOcm "github.com/konfidence-project/konfidence/pkg/ocm/repository"
)

type VectorPort interface {
	GetLatestVectorVersion(ctx context.Context, registryAndComponent string) (string, error)
}

type VectorPortProvider interface {
	NewVectorPort(verifier crypto.Verifier, client pkgOcm.Client) VectorPort
}

type VectorPortProviderFunc func(verifier crypto.Verifier, client pkgOcm.Client) VectorPort

func (f VectorPortProviderFunc) NewVectorPort(verifier crypto.Verifier, client pkgOcm.Client) VectorPort {
	return f(verifier, client)
}

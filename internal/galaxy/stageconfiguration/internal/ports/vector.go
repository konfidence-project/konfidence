package ports

//go:generate go run go.uber.org/mock/mockgen -destination=mocks/mock_vector_port.go -package=mocks github.com/konfidence-project/konfidence/internal/galaxy/stageconfiguration/internal/ports VectorPort

import (
	"context"

	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	pkgocm "github.com/konfidence-project/konfidence/pkg/ocm/repository"
)

type VectorPort interface {
	GetLatestVectorVersion(ctx context.Context, registryAndComponent string) (string, error)
}

type VectorPortProvider interface {
	NewVectorPort(verifier crypto.Verifier, client pkgocm.Client) VectorPort
}

type VectorPortProviderFunc func(verifier crypto.Verifier, client pkgocm.Client) VectorPort

func (f VectorPortProviderFunc) NewVectorPort(verifier crypto.Verifier, client pkgocm.Client) VectorPort {
	return f(verifier, client)
}

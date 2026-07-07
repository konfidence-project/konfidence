package ports

//go:generate go run go.uber.org/mock/mockgen -destination=mocks/mock_vector_port.go -package=mocks github.com/konfidence-project/konfidence/internal/galaxy/stageconfiguration/internal/ports VectorPort

import "context"

type VectorPort interface {
	GetLatestVectorVersion(ctx context.Context, registryAndComponent string) (string, error)
}

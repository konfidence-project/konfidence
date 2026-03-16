package domain

import "context"

type VectorPort interface {
	GetLatestVectorVersion(ctx context.Context, registryAndComponent string) (string, error)
}

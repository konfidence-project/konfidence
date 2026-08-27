package vectorpromotion

import (
	"context"
	"errors"
	"fmt"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var ErrVectorPromotionNotFound = errors.New("vectorPromotion not found")

type Repository interface {
	Get(ctx context.Context, projectId string, vectorPromotionId string) (*konfidence.VectorPromotion, error)
	Approve(ctx context.Context, projectId string, vectorPromotionId string) error
}

type k8sRepository struct{ k8sClient client.Client }

func NewRepository(k8sClient client.Client) Repository {
	return &k8sRepository{k8sClient: k8sClient}
}

func (r *k8sRepository) Get(ctx context.Context, projectId string, configId string) (*konfidence.VectorPromotion, error) {
	// TODO get vector promotion
	var vectorPromotion konfidence.VectorPromotion
	if err := r.k8sClient.Get(ctx, types.NamespacedName{Name: configId}, &vectorPromotion); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, ErrVectorPromotionNotFound
		}
		return nil, fmt.Errorf("getting vectorPromotion failed %q: %w", configId, err)
	}

	return &vectorPromotion, nil
}

func (r *k8sRepository) Approve(ctx context.Context, projectId string, vectorPromotionId string) error {
	// TODO implement approve

	return nil
}

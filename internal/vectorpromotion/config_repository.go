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

var ErrVectorPromotionConfigNotFound = errors.New("vectorPromotionConfig not found")

type ConfigRepository interface {
	Get(ctx context.Context, namespace string, vectorPromotionConfigId string) (*konfidence.VectorPromotionConfig, error)
	List(ctx context.Context, namespace string) ([]konfidence.VectorPromotionConfig, error)
}

type k8sConfigRepository struct{ reader client.Reader }

func NewConfigRepository(reader client.Reader) ConfigRepository {
	return &k8sConfigRepository{reader: reader}
}

func (r *k8sConfigRepository) Get(ctx context.Context, namespace string, configId string) (*konfidence.VectorPromotionConfig, error) {
	var vectorPromotionConfig konfidence.VectorPromotionConfig
	if err := r.reader.Get(ctx, types.NamespacedName{Namespace: namespace, Name: configId}, &vectorPromotionConfig); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, ErrVectorPromotionConfigNotFound
		}
		return nil, fmt.Errorf("getting vectorPromotionConfig failed %q: %w", configId, err)
	}

	return &vectorPromotionConfig, nil
}

func (r *k8sConfigRepository) List(ctx context.Context, namespace string) ([]konfidence.VectorPromotionConfig, error) {
	var configs konfidence.VectorPromotionConfigList
	if err := r.reader.List(ctx, &configs, client.InNamespace(namespace)); err != nil {
		return nil, fmt.Errorf("listing vectorPromotionConfigs in namespace %q failed: %w", namespace, err)
	}

	return configs.Items, nil
}

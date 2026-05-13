package controller

import (
	"context"

	global "github.com/konfidence-project/crds/api/global/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getPromotionConfig(ctx context.Context, clusterClient client.Client, vectorPromotion *global.VectorPromotion) (*global.VectorPromotionConfig, error) {
	config := &global.VectorPromotionConfig{}
	if err := clusterClient.Get(ctx, types.NamespacedName{
		Name:      vectorPromotion.Spec.VectorPromotionConfigRef,
		Namespace: vectorPromotion.Namespace,
	}, config); err != nil {
		return nil, err
	}
	return config, nil
}

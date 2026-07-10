package controller

import (
	"context"

	galaxy "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getPromotionConfig(ctx context.Context, clusterClient client.Client, vectorPromotion *galaxy.VectorPromotion) (*galaxy.VectorPromotionConfig, error) {
	config := &galaxy.VectorPromotionConfig{}
	if err := clusterClient.Get(ctx, types.NamespacedName{
		Name:      vectorPromotion.Spec.VectorPromotionConfigRef,
		Namespace: vectorPromotion.Namespace,
	}, config); err != nil {
		return nil, err
	}
	return config, nil
}

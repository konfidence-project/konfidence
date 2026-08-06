package controller

import (
	"context"
	"fmt"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getPromotionConfig(
	ctx context.Context, clusterClient client.Client, vectorPromotion *konfidence.VectorPromotion,
) (*konfidence.VectorPromotionConfig, error) {
	config := &konfidence.VectorPromotionConfig{}
	if err := clusterClient.Get(ctx, types.NamespacedName{
		Name:      vectorPromotion.Spec.VectorPromotionConfigRef,
		Namespace: vectorPromotion.Namespace,
	}, config); err != nil {
		return nil, err
	}
	return config, nil
}

// listSiblingPromotions returns all promotions in the promotion's namespace
// that reference the same VectorPromotionConfig, including the promotion
// itself. The field selector is served by the manager cache index
// (RegisterFieldIndexes) for cached clients and by the CRD's selectableFields
// for direct clients.
func listSiblingPromotions(
	ctx context.Context, clusterClient client.Client, vectorPromotion *konfidence.VectorPromotion,
) ([]konfidence.VectorPromotion, error) {
	return listPromotionsForConfig(ctx, clusterClient, vectorPromotion.Namespace, vectorPromotion.Spec.VectorPromotionConfigRef)
}

// listPromotionsForConfig returns all promotions in the namespace referencing
// the named VectorPromotionConfig.
func listPromotionsForConfig(
	ctx context.Context, clusterClient client.Client, namespace, configName string,
) ([]konfidence.VectorPromotion, error) {
	list := &konfidence.VectorPromotionList{}
	err := clusterClient.List(ctx, list,
		client.InNamespace(namespace),
		client.MatchingFields{promotionConfigRefField: configName})
	if err != nil {
		return nil, fmt.Errorf("failed to list promotions of config %q: %w", configName, err)
	}
	return list.Items, nil
}

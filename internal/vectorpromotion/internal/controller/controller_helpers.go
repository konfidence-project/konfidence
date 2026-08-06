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
// that reference the same VectorPromotionConfig, including the promotion itself.
func listSiblingPromotions(
	ctx context.Context, clusterClient client.Client, vectorPromotion *konfidence.VectorPromotion,
) ([]konfidence.VectorPromotion, error) {
	list := &konfidence.VectorPromotionList{}
	if err := clusterClient.List(ctx, list, client.InNamespace(vectorPromotion.Namespace)); err != nil {
		return nil, fmt.Errorf("failed to list sibling promotions: %w", err)
	}
	siblings := make([]konfidence.VectorPromotion, 0, len(list.Items))
	for _, item := range list.Items {
		if item.Spec.VectorPromotionConfigRef == vectorPromotion.Spec.VectorPromotionConfigRef {
			siblings = append(siblings, item)
		}
	}
	return siblings, nil
}

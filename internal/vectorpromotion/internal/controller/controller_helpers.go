package controller

import (
	"context"
	"fmt"
	"slices"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getPromotionConfig(
	ctx context.Context, clusterClient client.Reader, vectorPromotion *konfidence.VectorPromotion,
) (*konfidence.VectorPromotionConfig, error) {
	config := &konfidence.VectorPromotionConfig{}
	if err := clusterClient.Get(ctx, types.NamespacedName{
		Name:      vectorPromotion.Spec.VectorPromotionConfigName,
		Namespace: vectorPromotion.Namespace,
	}, config); err != nil {
		return nil, err
	}
	return config, nil
}

// listSiblingPromotions excludes self so a cached, pre-patch copy cannot
// override the status already evaluated by the current reconciliation.
func listSiblingPromotions(ctx context.Context, c client.Reader, current *konfidence.VectorPromotion) ([]konfidence.VectorPromotion, error) {
	promotions, err := listPromotionsForConfig(ctx, c, current.Namespace, current.Spec.VectorPromotionConfigName)
	if err != nil {
		return nil, err
	}
	return slices.DeleteFunc(promotions, func(p konfidence.VectorPromotion) bool {
		return p.UID == current.UID
	}), nil
}

// listPromotionsForConfig returns all promotions in the namespace referencing
// the named VectorPromotionConfig.
func listPromotionsForConfig(
	ctx context.Context, clusterClient client.Reader, namespace, configName string,
) ([]konfidence.VectorPromotion, error) {
	list := &konfidence.VectorPromotionList{}
	err := clusterClient.List(ctx, list,
		client.InNamespace(namespace),
		client.MatchingFields{promotionConfigNameField: configName})
	if err != nil {
		return nil, fmt.Errorf("failed to list promotions of config %q: %w", configName, err)
	}
	return list.Items, nil
}

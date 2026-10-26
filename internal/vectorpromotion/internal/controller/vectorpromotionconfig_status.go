package controller

import (
	"context"
	"fmt"
	"reflect"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/konfidence-project/konfidence/internal/vectorpromotion/internal/promotion"
)

// patchConfigStatus applies mutate to the config status and patches it if it
// changed. Plain merge patch is safe because this reconciler is the only
// config status writer and controller-runtime serializes reconciles of the
// same config within it.
func (r *VectorPromotionConfigReconciler) patchConfigStatus(ctx context.Context, config *konfidence.VectorPromotionConfig, mutate func()) error {
	original := config.DeepCopy()
	mutate()
	if equalConfigStatus(config, original) {
		return nil
	}
	if err := r.Status().Patch(ctx, config, client.MergeFrom(original)); err != nil {
		return fmt.Errorf("failed to patch status of VectorPromotionConfig %q in namespace %q: %w",
			config.Name, config.Namespace, err)
	}
	return nil
}

// setConfigReadyCondition writes the config's Ready condition, telling users
// whether the resources their config references actually exist.
func setConfigReadyCondition(
	config *konfidence.VectorPromotionConfig,
	status metav1.ConditionStatus,
	reason, message string,
) {
	meta.SetStatusCondition(&config.Status.Conditions, metav1.Condition{
		Type:               konfidence.VectorPromotionConfigReadyCondition,
		Status:             status,
		ObservedGeneration: config.Generation,
		Reason:             reason,
		Message:            message,
	})
}

func equalConfigStatus(a, b *konfidence.VectorPromotionConfig) bool {
	return reflect.DeepEqual(a.Status, b.Status)
}

// aggregatePromotionResults mirrors the newest promotion's conditions onto
// the config's last-promotion view, Deployment-style: the config owns its
// promotions and recomputes the aggregate from the full list, so the result
// is independent of event ordering. With no promotions left (e.g. after
// retention reaping) the last known views are kept.
func aggregatePromotionResults(config *konfidence.VectorPromotionConfig, promotions []konfidence.VectorPromotion) {
	if newest := promotion.Newest(promotions); newest != nil && len(newest.Status.Conditions) > 0 {
		config.Status.LastPromotionConditions = newest.Status.Conditions
	}
	succeeded := make([]konfidence.VectorPromotion, 0, len(promotions))
	for i := range promotions {
		if promotion.IsSucceeded(&promotions[i]) {
			succeeded = append(succeeded, promotions[i])
		}
	}
	if newestSucceeded := promotion.Newest(succeeded); newestSucceeded != nil {
		config.Status.LastSuccessfulPromotionConditions = newestSucceeded.Status.Conditions
	}
}

// promotionName builds `<config>-<sequence>`, trimming the config part when
// the result would exceed the DNS subdomain limit.
func promotionName(configName string, sequence int64) string {
	suffix := fmt.Sprintf("-%d", sequence)
	if len(configName)+len(suffix) > 253 {
		configName = configName[:253-len(suffix)]
	}
	return configName + suffix
}

package controller

import (
	"context"
	"fmt"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// patchConfigStatus applies mutate to the config status and patches it if it
// changed. Plain merge patch: this reconciler and the execution controller
// write disjoint config status fields (conditions/sequence here,
// lastPromotion* there), and controller-runtime serializes reconciles of the
// same config within this controller.
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
	if a.Status.Sequence != b.Status.Sequence {
		return false
	}
	current := meta.FindStatusCondition(a.Status.Conditions, konfidence.VectorPromotionConfigReadyCondition)
	previous := meta.FindStatusCondition(b.Status.Conditions, konfidence.VectorPromotionConfigReadyCondition)
	if current == nil || previous == nil {
		return current == previous
	}
	return current.Status == previous.Status && current.Reason == previous.Reason && current.Message == previous.Message
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

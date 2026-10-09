package controller

import (
	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// setReadyConditionFalse records which step the vector deployment is waiting on. Writing it on every incomplete
// pass resets a Ready=True from earlier, so a vector whose ArtifactDeployment later stalls is never both Ready
// and Stalled.
func setReadyConditionFalse(vectorDeployment *konfidence.VectorDeployment, reason, message string) {
	meta.SetStatusCondition(&vectorDeployment.Status.Conditions, metav1.Condition{
		Type:               konfidence.VectorReadyCondition,
		Status:             metav1.ConditionFalse,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: vectorDeployment.Generation,
	})
}

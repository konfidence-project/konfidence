/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	// StageSyncKind is kind of the StageSync resource.
	StageSyncKind = "StageSync"

	// StageSyncAppliedCondition is the applied condition for the StageSync resource.
	// It indicates whether the Stage defined in the StageSync resource has been successfully applied to the LCP cluster.
	StageSyncAppliedCondition = "Applied"
	// StageDeletedCondition tracks the deletion lifecycle of the local Stage.
	StageDeletedCondition = "StageDeleted"

	// StageReconcileSuccessfulReason indicates that the stage was successfully reconciled on the LCP cluster.
	StageReconcileSuccessfulReason = "StageReconcileSuccessful"
	// StageReconcileFailedReason indicates that the stage reconciliation on the LCP cluster failed.
	StageReconcileFailedReason = "StageReconcileFailed"
	// NamespaceNotFoundReason indicates that the target namespace for the stage was not found on the LCP cluster.
	NamespaceNotFoundReason = "NamespaceNotFound"
	// StageQueryFailedReason indicates that the stage defined in the StageSync resource could not be queried from the LCP cluster.
	StageQueryFailedReason = "StageQueryFailed"
	// ConflictWithUnmanagedStageReason indicates that there is a conflict with an unmanaged stage on the LCP cluster.
	ConflictWithUnmanagedStageReason = "ConflictWithUnmanagedStage"
	// AddingFinalizerFailedReason indicates that adding the finalizer to the StageSync resource failed.
	AddingFinalizerFailedReason = "AddingFinalizerFailed"
	// StageDeletionFailedReason indicates that the stage deletion on the LCP cluster failed.
	StageDeletionFailedReason = "StageDeletionFailed"
	// RemovingFinalizerFailedReason indicates that removing the finalizer from the StageSync resource failed.
	RemovingFinalizerFailedReason = "RemovingFinalizerFailed"
	// StageCrdQueryFailedReason indicates that the Stage CRD could not be queried from the LCP cluster.
	StageCrdQueryFailedReason = "StageCrdQueryFailed"
	// APIVersionNotSupportedReason indicates that the Stage CRD on the LCP cluster does not support the requested API version.
	APIVersionNotSupportedReason = "APIVersionNotSupported"
	// InvalidStageTemplateReason indicates that the stage template defined in the StageSync resource is invalid.
	InvalidStageTemplateReason = "InvalidStageTemplate"
	// StageStatusReflectionFailedReason indicates that the stage status could not be reflected back to the StageSync resource.
	StageStatusReflectionFailedReason = "StageStatusReflectionFailed"
	// StageDeletionInitiatedReason indicates the Stage delete call has been issued
	// but the object has not yet been confirmed as removed.
	StageDeletionInitiatedReason = "StageDeletionInitiated"
	// StageDeletionBlockedReason indicates the Stage still exists and has a non-zero
	// DeletionTimestamp, meaning it is stuck on one or more finalizers.
	StageDeletionBlockedReason = "StageDeletionBlocked"
	// StageDeletionSuccessfulReason is set on the StageDeleted condition when the
	// local Stage has been confirmed gone and the StageSync finalizer is released.
	StageDeletionSuccessfulReason = "StageDeletionSuccessful"
)

// StageSyncSpec defines the desired state of a StageSync.
type StageSyncSpec struct {
	// StageTemplate contains the template of the stage to be created on the LCP cluster.
	StageTemplate runtime.RawExtension `json:"stageTemplate"`

	// ReconcileInterval defines how often the sync controller should reconcile the StageSync resource.
	// If not set, the controller's default reconcile interval will be used.
	// +kubebuilder:validation:Optional
	ReconcileInterval *metav1.Duration `json:"reconcileInterval,omitempty"`
}

// StageSyncStatus defines the observed state of a StageSync.
type StageSyncStatus struct {
	Conditions  []metav1.Condition   `json:"conditions,omitempty"`
	StageStatus runtime.RawExtension `json:"stageStatus,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Applied",type=string,JSONPath=".status.conditions[?(@.type=='Applied')].status",description="Applied status"
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=".metadata.creationTimestamp",description="Age"

// StageSync is the Schema for the stageSyncs API.
type StageSync struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StageSyncSpec   `json:"spec,omitempty"`
	Status StageSyncStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// StageSyncList contains a list of StageSync.
type StageSyncList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StageSync `json:"items"`
}

func init() {
	SchemeBuilder.Register(&StageSync{}, &StageSyncList{})
}

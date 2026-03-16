package template

import (
	common "github.com/konfidence-project/crds/api/common/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// StageTemplate is used to initialize a StageSync object
type StageTemplate struct {
	metav1.TypeMeta `json:",inline"`
	Metadata        types.NamespacedName `json:"metadata,omitempty"`
	Spec            common.StageSpec     `json:"spec,omitempty"`
}

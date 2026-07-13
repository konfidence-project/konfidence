package template

import (
	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StageTemplate is used to initialize a StageSync object
type StageTemplate struct {
	metav1.TypeMeta `json:",inline"`
	Metadata        NamespacedName       `json:"metadata"`
	Spec            konfidence.StageSpec `json:"spec"`
}

type NamespacedName struct {
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name"`
}

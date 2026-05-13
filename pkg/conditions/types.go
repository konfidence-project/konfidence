package conditions

//go:generate go run go.uber.org/mock/mockgen -source=types.go -destination=internal/mocks/mock_types_port.go -package=mocks

import (
	metav1 "k8s.io/apimachinery/pkg/api/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Getter is an interface that combines client.Object and v1alpha1.ConditionGetter.
// It is used to access Kubernetes objects that support reading conditions.
type Getter interface {
	client.Object
	GetConditions() []*metav1.Condition
}

// Setter is an interface that combines client.Object, v1alpha1.ConditionGetter, and v1alpha1.ConditionSetter.
// It is used to access and modify Kubernetes objects that support reading and writing conditions.
type Setter interface {
	Getter
	SetConditions(conditions []*metav1.Condition)
}

// ConditionType represents a condition in the status of a resource.
type ConditionType string

// A set of common condition types
const (

	// ConditionReady indicates that the resource is ready.
	ConditionReady ConditionType = "Ready"
)

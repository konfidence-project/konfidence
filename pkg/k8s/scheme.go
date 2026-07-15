package k8s

import (
	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
)

// NewScheme returns a runtime.Scheme with all Konfidence CRDs (galaxy + star)
// and the core Kubernetes types registered.
//
// Both the API server and the kden CLI use this scheme so that any
// client.Client built from it can read and write the full Konfidence resource
// space without additional registration steps in each binary.
func NewScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(s))
	utilruntime.Must(konfidence.AddToScheme(s))
	return s
}

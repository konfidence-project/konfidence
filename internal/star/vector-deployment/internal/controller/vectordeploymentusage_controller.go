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

package controller

import (
	"context"
	"crypto/sha256"
	"fmt"

	landscape "github.tools.sap/konfidence/crds/api/landscape/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// VectorDeploymentUsageReconciler reconciles a VectorDeploymentUsage object
type VectorDeploymentUsageReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=landscape.konfidence.tools.sap,resources=vectordeploymentusages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=landscape.konfidence.tools.sap,resources=vectordeploymentusages/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=landscape.konfidence.tools.sap,resources=vectordeploymentusages/finalizers,verbs=update

// +kubebuilder:rbac:groups=landscape.konfidence.tools.sap,resources=vectordeployments,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the VectorDeploymentUsage object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *VectorDeploymentUsageReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Reconciling VectorDeploymentUsage")

	// get vector deployment usage
	vdu := &landscape.VectorDeploymentUsage{}
	if err := r.Get(ctx, req.NamespacedName, vdu); err != nil {
		log.Error(err, "Unable to fetch vector deployment usage")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	log.Info("Found vector deployment usage")

	vds := &landscape.VectorDeploymentList{}
	if err := r.List(ctx, vds, &client.ListOptions{Namespace: req.Namespace}); err != nil {
		log.Error(err, "Unable to list vector deployments")
		return ctrl.Result{}, err
	}
	log.Info("Listed vector deployments", "count", len(vds.Items))

	var found *landscape.VectorDeployment = nil
	for _, vd := range vds.Items {
		if vd.Spec.Vector == vdu.Spec.Vector {
			found = &vd
			break
		}
	}
	if found == nil {
		log.Info("No vector deployment found for the usage", "vector", vdu.Spec.Vector)
		found = r.constructVectorDeploymentForUsage(vdu)
		found.OwnerReferences = append(found.OwnerReferences, constructOwnerReference(vdu))

		if err := r.Create(ctx, found); err != nil {
			return ctrl.Result{}, err
		}
		log.Info("Created vector deployment for the usage", "vector", vdu.Spec.Vector, "name", found.Name)
	} else {
		log.Info("Found existing vector deployment for the usage", "vector", vdu.Spec.Vector, "name", found.Name)
		var ownerRef *metav1.OwnerReference = nil
		for _, ref := range found.OwnerReferences {
			if ref.UID == vdu.UID {
				ownerRef = &ref
				break
			}
		}
		if ownerRef == nil {
			log.Info("Adding owner reference to existing vector deployment", "vector", vdu.Spec.Vector, "name", found.Name)
			found.OwnerReferences = append(found.OwnerReferences, constructOwnerReference(vdu))
			if err := r.Update(ctx, found); err != nil {
				return ctrl.Result{}, err
			}
		} else {
			log.Info("Vector deployment already has owner reference", "vector", vdu.Spec.Vector, "name", found.Name)
		}
	}

	return ctrl.Result{}, nil
}

func constructOwnerReference(vdu *landscape.VectorDeploymentUsage) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: vdu.APIVersion,
		Kind:       vdu.Kind,
		Name:       vdu.Name,
		UID:        vdu.UID,
	}
}

func (r *VectorDeploymentUsageReconciler) constructVectorDeploymentForUsage(vdu *landscape.VectorDeploymentUsage) *landscape.VectorDeployment {
	h := sha256.New()
	h.Write([]byte(vdu.Spec.Vector))
	hash := h.Sum(nil)
	name := fmt.Sprintf("%x", hash)

	return &landscape.VectorDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
			Name:        name,
			Namespace:   vdu.Namespace,
		},
		Spec: landscape.VectorDeploymentSpec{
			Vector: vdu.Spec.Vector,
		},
		Status: landscape.VectorDeploymentStatus{},
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *VectorDeploymentUsageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&landscape.VectorDeploymentUsage{}).
		Owns(&landscape.VectorDeployment{}).
		Named("vectordeploymentusage").
		Complete(r)
}

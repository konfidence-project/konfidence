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

	landscape "github.tools.sap/konfidence/crds/api/landscape/v1alpha1"

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

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *VectorDeploymentUsageReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&landscape.VectorDeploymentUsage{}).
		Owns(&landscape.VectorDeployment{}).
		Named("vectordeploymentusage").
		Complete(r)
}

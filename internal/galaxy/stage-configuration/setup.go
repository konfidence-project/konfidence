/*
Copyright 2026.

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

package stageconfiguration

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"

	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"
)

// Options configures the stage configuration controllers.
type Options struct {
	// VectorVerifier is used to verify vector signatures.
	// If nil or a NoopVerifier, verification is disabled.
	VectorVerifier crypto.Verifier
}

// SetupControllers registers all stage configuration controllers with the given manager.
func SetupControllers(mgr mcmanager.Manager, scheme *runtime.Scheme, restConfig *rest.Config, opts Options) error {
	if err := NewStageConfigurationReconciler(
		mgr,
		scheme,
		restConfig,
		opts.VectorVerifier,
	).SetupWithManager(mgr); err != nil {
		return err
	}

	return nil
}

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

package vectorassembly

import (
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/konfidence-project/konfidence/internal/galaxy/vector-assembly/internal/controller"
	"github.com/konfidence-project/konfidence/internal/galaxy/vector-assembly/internal/controller/domain"
	"github.com/konfidence-project/konfidence/internal/galaxy/vector-assembly/pkg/ocm"
	"github.com/konfidence-project/konfidence/pkg/ocm/crypto"
	"github.com/konfidence-project/konfidence/pkg/ocm/repository"
	mcmanager "sigs.k8s.io/multicluster-runtime/pkg/manager"
)

// Options configures the vector assembly controllers.
type Options struct {
	// ArtifactVerifier is used to verify artifact signatures.
	// If nil, artifact verification is disabled.
	ArtifactVerifier crypto.Verifier

	// VectorVerifier is used to verify vector signatures (e.g. base vector).
	// If nil, vector verification is disabled.
	VectorVerifier crypto.Verifier

	// VectorSigner is used to sign newly assembled vectors.
	// If nil, vector signing is disabled.
	VectorSigner crypto.Signer
}

// SetupControllers registers all vector assembly controllers with the given manager.
func SetupControllers(mgr mcmanager.Manager, scheme *runtime.Scheme, opts Options) error {
	adapterConfig := []ocm.AdapterOption{
		ocm.WithArtifactVerifier(opts.ArtifactVerifier),
		ocm.WithVectorVerifier(opts.VectorVerifier),
		ocm.WithVectorSigner(opts.VectorSigner),
	}

	if err := (&controller.VectorTemplateReconciler{
		Mgr:                   mgr,
		Scheme:                scheme,
		OcmClientProvider:     repository.DefaultOciClientProvider,
		VectorOcmPortProvider: ocm.NewPortProvider(adapterConfig...),
		VersionGenerator:      domain.TimestampVectorVersionGenerator,
	}).SetupWithManager(mgr); err != nil {
		return err
	}

	return nil
}

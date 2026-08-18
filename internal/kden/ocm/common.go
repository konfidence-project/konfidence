package ocm

import (
	"context"

	ocmconstructor "ocm.software/open-component-model/bindings/go/constructor"
	ocmconstructorruntime "ocm.software/open-component-model/bindings/go/constructor/runtime"
	ocmcompref "ocm.software/open-component-model/bindings/go/oci/compref"
	ocmsigninghandler "ocm.software/open-component-model/bindings/go/plugin/manager/registries/signinghandler"
	ocmruntime "ocm.software/open-component-model/bindings/go/runtime"
	ocmsigning "ocm.software/open-component-model/bindings/go/signing"
)

var (
	ocmGetPluginManager               = GetPluginManager
	ocmGetRepositorySpec              = GetRepositorySpec
	ocmGetComponentRepositoryResolver = GetComponentRepositoryResolver
	ocmGetCredentialGraph             = GetCredentialGraph
	ocmConvertToRuntimeConstructor    = ocmconstructorruntime.ConvertToRuntimeConstructor
	ocmCreateConstructor              = ocmconstructor.NewDefaultConstructor
	ocmNewComponentRepositoryResolver = NewComponentRepositoryResolver
	ocmGenerateDigestForSigning       = ocmsigning.GenerateDigest
	ocmParseComponentReference        = ocmcompref.Parse
	ocmGetTargetRepository            = GetTargetRepository
	ocmGetSigningHandler              = func(ctx context.Context, registry *ocmsigninghandler.SigningRegistry, spec ocmruntime.Typed) (ocmsigning.Handler, error) {
		return registry.GetPlugin(ctx, spec)
	}
)

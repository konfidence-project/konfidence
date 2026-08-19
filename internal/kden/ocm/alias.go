package ocm

import (
	"context"
	"fmt"

	ocmgenericspecv1 "ocm.software/open-component-model/bindings/go/configuration/generic/v1/spec"
	ocmoci "ocm.software/open-component-model/bindings/go/oci"
	ocmcompref "ocm.software/open-component-model/bindings/go/oci/compref"
	ocmrepositoryctfv1 "ocm.software/open-component-model/bindings/go/oci/spec/repository/v1/ctf"
)

// AliasProperties holds the parameters for creating or updating an alias tag.
type AliasProperties struct {
	// ComponentVersion is the full ref string: registry//component:version
	ComponentVersion string
	// Alias is the mutable tag to create or update (e.g. "edge", "latest").
	Alias string
}

// Alias creates or updates a mutable alias tag pointing to the given component version.
func Alias(ctx context.Context, props AliasProperties, ocmConfiguration *ocmgenericspecv1.Config) error {
	pluginManager, err := ocmGetPluginManager(ctx, "")
	if err != nil {
		return fmt.Errorf("failed to get plugin manager: %w", err)
	}

	credentials, err := ocmGetCredentialGraph(ctx, pluginManager, ocmConfiguration)
	if err != nil {
		return fmt.Errorf("failed to get credential graph: %w", err)
	}

	componentReference, err := ocmParseComponentReference(props.ComponentVersion, ocmcompref.WithCTFAccessMode(ocmrepositoryctfv1.AccessModeReadWrite))
	if err != nil {
		return fmt.Errorf("failed to parse component version: %w", err)
	}

	repoResolver, err := ocmNewComponentRepositoryResolver(ctx, pluginManager.ComponentVersionRepositoryRegistry,
		credentials, WithConfig(ocmConfiguration), WithComponentRef(componentReference))
	if err != nil {
		return fmt.Errorf("failed to initialize ocm repository: %w", err)
	}

	repo, err := repoResolver.GetComponentVersionRepositoryForComponent(ctx, componentReference.Component, componentReference.Version)
	if err != nil {
		return fmt.Errorf("failed to access ocm repository: %w", err)
	}

	aliasRepo, ok := repo.(ocmoci.AliasComponentVersionRepository)
	if !ok {
		return fmt.Errorf("repository does not support aliasing")
	}

	return aliasRepo.AddComponentVersionAlias(ctx, componentReference.Component, componentReference.Version, props.Alias)
}

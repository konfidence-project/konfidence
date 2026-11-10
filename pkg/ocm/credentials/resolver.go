package credentials

import (
	"context"
	"fmt"

	konfidence "github.com/konfidence-project/konfidence/api/v1alpha1"
	"ocm.software/open-component-model/bindings/go/credentials"
	credentialsruntime "ocm.software/open-component-model/bindings/go/credentials/spec/config/runtime"
	ocmv1alpha1 "ocm.software/open-component-model/bindings/go/kubernetes/controller/api/v1alpha1"
	"ocm.software/open-component-model/bindings/go/kubernetes/controller/pkg/configuration"
	ocicredentials "ocm.software/open-component-model/bindings/go/oci/credentials"
	ociidentityv1 "ocm.software/open-component-model/bindings/go/oci/spec/identity/v1"
	"ocm.software/open-component-model/bindings/go/plugin/manager"
	rsacredentials "ocm.software/open-component-model/bindings/go/rsa/spec/credentials"
	"ocm.software/open-component-model/bindings/go/runtime"
	s3credentials "ocm.software/open-component-model/bindings/go/s3/spec/credentials/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Ref is a reference to a Secret holding credential data (.ocmconfig or .dockerconfigjson).
type Ref struct {
	Name string
}

// ResolverFromRefs builds a credentials.Resolver by loading and flat-merging all
// referenced Secrets in namespace into a single credential graph.
// Returns (nil, nil) if refs is empty.
func ResolverFromRefs(
	ctx context.Context,
	k8sClient client.Reader,
	namespace string,
	refs []Ref,
) (credentials.Resolver, error) {
	if len(refs) == 0 {
		return nil, nil
	}

	ocmConfigs := make([]ocmv1alpha1.OCMConfiguration, len(refs))
	for i, ref := range refs {
		ocmConfigs[i] = ocmv1alpha1.OCMConfiguration{
			NamespacedObjectKindReference: ocmv1alpha1.NamespacedObjectKindReference{
				Kind:      "Secret",
				Name:      ref.Name,
				Namespace: namespace,
			},
		}
	}

	ocmCfg, err := configuration.LoadConfigurations(ctx, k8sClient, namespace, ocmConfigs)
	if err != nil {
		return nil, fmt.Errorf("load OCM configurations: %w", err)
	}

	cfg, err := credentialsruntime.LookupCredentialConfig(ocmCfg.Config)
	if err != nil {
		return nil, fmt.Errorf("lookup credential config: %w", err)
	}
	if cfg == nil {
		return nil, fmt.Errorf("no credential configuration found in OCM config")
	}

	return buildGraph(ctx, cfg)
}

// ResolverFromCredentials translates *konfidence.Credentials to []Ref and
// delegates to ResolverFromRefs.
// Returns (nil, nil) if creds is nil or creds.OCM is nil.
func ResolverFromCredentials(
	ctx context.Context,
	k8sClient client.Reader,
	namespace string,
	creds *konfidence.Credentials,
) (credentials.Resolver, error) {
	if creds == nil || creds.OCM == nil {
		return nil, nil
	}
	refs := make([]Ref, len(creds.OCM.Refs))
	for i, r := range creds.OCM.Refs {
		refs[i] = Ref{Name: r.Name}
	}
	return ResolverFromRefs(ctx, k8sClient, namespace, refs)
}

// credentialTypeScheme is a CredentialTypeSchemeProvider that tells the credential
// graph to treat the registered credential types as first-class typed credentials
// (resolved directly from the graph) rather than routing them through the plugin path.
type credentialTypeScheme struct{ scheme *runtime.Scheme }

func (r *credentialTypeScheme) GetCredentialTypeScheme() *runtime.Scheme { return r.scheme }

func buildGraph(ctx context.Context, cfg *credentialsruntime.Config) (credentials.Resolver, error) {
	pm := manager.NewPluginManager(ctx)
	// The OCI plugin is required solely for .dockerconfigjson secrets: LoadConfigurations converts
	// them into a credentials.config/v1 with a repositories: entry (DockerConfig/v1) rather than a
	// consumers: entry, so there are no direct graph edges — resolution falls through to the
	// repository plugin path. .ocmconfig secrets with explicit consumers: sections (OCI or RSA)
	// resolve via the direct graph path and need no plugin at all. All sources are flat-merged
	// into this single graph, which can be shared across OCM related clients.
	if err := pm.CredentialRepositoryRegistry.RegisterInternalCredentialRepositoryPlugin(
		&ocicredentials.OCICredentialRepository{}, []runtime.Type{ociidentityv1.Type}); err != nil {
		return nil, fmt.Errorf("register OCI credential repository plugin: %w", err)
	}

	// RSACredentials/v1 and S3Credentials/v1 must be registered in the credential
	// type scheme so that extractResolvable treats them as direct typed credentials.
	// Without this, the graph builder routes them to the (nil) credential plugin
	// provider and panics.
	credTypeScheme := runtime.NewScheme()
	rsacredentials.MustRegisterCredentialType(credTypeScheme)
	s3credentials.MustRegisterCredentialType(credTypeScheme)

	graph, err := credentials.ToGraph(ctx, cfg, credentials.Options{
		CredentialRepositoryTypeScheme: pm.CredentialRepositoryRegistry.RepositoryScheme(),
		RepositoryPluginProvider:       pm.CredentialRepositoryRegistry,
		CredentialPluginProvider:       pm.CredentialPluginRegistry,
		CredentialTypeSchemeProvider:   &credentialTypeScheme{scheme: credTypeScheme},
	})
	if err != nil {
		return nil, fmt.Errorf("create credential graph: %w", err)
	}
	return graph, nil
}

package credentials

import (
	"context"
	"fmt"

	galaxy "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	"ocm.software/open-component-model/bindings/go/credentials"
	credentialsruntime "ocm.software/open-component-model/bindings/go/credentials/spec/config/runtime"
	ocicredentials "ocm.software/open-component-model/bindings/go/oci/credentials"
	ociidentityv1 "ocm.software/open-component-model/bindings/go/oci/spec/identity/v1"
	"ocm.software/open-component-model/bindings/go/plugin/manager"
	rsacredentials "ocm.software/open-component-model/bindings/go/rsa/spec/credentials"
	"ocm.software/open-component-model/bindings/go/runtime"
	ocmv1alpha1 "ocm.software/open-component-model/kubernetes/controller/api/v1alpha1"
	"ocm.software/open-component-model/kubernetes/controller/pkg/configuration"
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

// ResolverFromCredentials is the galaxy domain mapper — it translates *galaxy.Credentials
// to []Ref and delegates to ResolverFromRefs.
// Returns (nil, nil) if creds is nil or creds.OCM is nil.
func ResolverFromCredentials(
	ctx context.Context,
	k8sClient client.Reader,
	namespace string,
	creds *galaxy.Credentials,
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

// rsaCredentialTypeScheme is a CredentialTypeSchemeProvider that tells the credential
// graph to treat RSACredentials/v1 as a first-class typed credential (resolved directly
// from the graph) rather than routing it through the plugin path.
type rsaCredentialTypeScheme struct{ scheme *runtime.Scheme }

func (r *rsaCredentialTypeScheme) GetCredentialTypeScheme() *runtime.Scheme { return r.scheme }

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

	// RSACredentials/v1 must be registered in the credential type scheme so that
	// extractResolvable treats it as a direct typed credential. Without this, the
	// graph builder routes it to the (nil) credential plugin provider and panics.
	credTypeScheme := runtime.NewScheme()
	rsacredentials.MustRegisterCredentialType(credTypeScheme)

	graph, err := credentials.ToGraph(ctx, cfg, credentials.Options{
		CredentialRepositoryTypeScheme: pm.CredentialRepositoryRegistry.RepositoryScheme(),
		RepositoryPluginProvider:       pm.CredentialRepositoryRegistry,
		CredentialPluginProvider:       pm.CredentialPluginRegistry,
		CredentialTypeSchemeProvider:   &rsaCredentialTypeScheme{scheme: credTypeScheme},
	})
	if err != nil {
		return nil, fmt.Errorf("create credential graph: %w", err)
	}
	return graph, nil
}

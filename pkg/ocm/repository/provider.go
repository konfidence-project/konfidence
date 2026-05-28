package repository

//go:generate go run go.uber.org/mock/mockgen -source=provider.go -destination=internal/mocks/mock_client_provider.go -package=mocks

import (
	"context"
	"fmt"

	galaxy "github.com/konfidence-project/konfidence/api/galaxy/v1alpha1"
	"ocm.software/open-component-model/kubernetes/controller/api/v1alpha1"
	"ocm.software/open-component-model/kubernetes/controller/pkg/configuration"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// ClientProvider creates configured Client instances for accessing OCM repositories.
// It encapsulates credential resolution, allowing controllers to obtain authenticated
// clients without knowing how credentials are sourced or configured.
type ClientProvider interface {
	NewClient(ctx context.Context, k8sClient client.Reader, namespace string, credentialsConfig []galaxy.CredentialsConfig) (Client, error)
}

type ClientProviderFunc func(ctx context.Context, k8sClient client.Reader, namespace string, credentialsConfig []galaxy.CredentialsConfig) (Client, error)

func (f ClientProviderFunc) NewClient(
	ctx context.Context, k8sClient client.Reader, namespace string, credentialsConfig []galaxy.CredentialsConfig,
) (Client, error) {
	return f(ctx, k8sClient, namespace, credentialsConfig)
}

var DefaultOciClientProvider = ClientProviderFunc(
	func(ctx context.Context, k8sClient client.Reader, namespace string, credentialsConfig []galaxy.CredentialsConfig) (Client, error) {
		log := logf.FromContext(ctx)

		ocmConfigs := mapToOCMConfigurations(credentialsConfig, namespace)

		ocmCfg, err := configuration.LoadConfigurations(ctx, k8sClient, namespace, ocmConfigs)
		if err != nil {
			return nil, fmt.Errorf("error loading ocm configuration: %w", err)
		}

		return NewOciClientBuilder().WithLogger(log).WithOCMConfig(ocmCfg).Build(ctx)
	})

func mapToOCMConfigurations(credentialsConfig []galaxy.CredentialsConfig, namespace string) []v1alpha1.OCMConfiguration {
	ocmConfigs := make([]v1alpha1.OCMConfiguration, len(credentialsConfig))
	for i, credConfig := range credentialsConfig {
		ocmConfigs[i] = v1alpha1.OCMConfiguration{
			NamespacedObjectKindReference: v1alpha1.NamespacedObjectKindReference{
				APIVersion: credConfig.APIVersion,
				Kind:       credConfig.Kind,
				Name:       credConfig.Name,
				Namespace:  namespace,
			},
			Policy: "", // TODO: what policy do konfidence system want to set here?
		}
	}
	return ocmConfigs
}

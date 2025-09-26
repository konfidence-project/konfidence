package ocm

import (
	"context"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"ocm.software/ocm/api/oci"
	"ocm.software/ocm/api/ocm"
	"ocm.software/ocm/api/tech/oci/identity"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ContextProvider interface to create an OCM Context
type ContextProvider interface {
	GetOCMContext(ctx context.Context) (ocm.Context, error)
}

type Provider struct {
	kubeClient client.Client // todo: remove KubeClient with proper configuration/credential struct
}

var _ ContextProvider = (*Provider)(nil)

func NewOCMContextProvider(kubeClient client.Client) *Provider {
	return &Provider{
		kubeClient: kubeClient,
	}
}

// GetOCMContext creates and returns an OCM context populated with credentials.
func (p *Provider) GetOCMContext(ctx context.Context) (ocm.Context, error) {
	ocmCtx := ocm.DefaultContext()

	// todo: create this namespace manually for now, later we need a proper configuration management
	konfidenceNamespace := "konfidence-system"

	secretList := v1.SecretList{}
	if err := p.kubeClient.List(ctx, &secretList, client.InNamespace(konfidenceNamespace)); err != nil {
		return nil, errors.Wrapf(err, "failed to list secrets in namespace %q", konfidenceNamespace)
	}
	for _, secret := range secretList.Items {
		if secret.Type != v1.SecretTypeDockerConfigJson {
			continue
		}
		err := parseDockerConfigJsonSecret(secret, ocmCtx)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse secret %q", secret.Name)
		}
	}

	return ocmCtx, nil
}

func parseDockerConfigJsonSecret(s v1.Secret, ocmCtx ocm.Context) error {
	dockerConfigJson, ok := s.Data[v1.DockerConfigJsonKey]
	if !ok {
		return errors.Errorf("secret %q does not contain key %q", s.Name, v1.DockerConfigJsonKey)
	}
	var dockerConfig configfile.ConfigFile
	if err := json.Unmarshal(dockerConfigJson, &dockerConfig); err != nil {
		return errors.Wrapf(err, "failed to unmarshal docker config json from secret %q", s.Name)
	}
	for registry, authConfig := range dockerConfig.AuthConfigs {
		creds := identity.SimpleCredentials(authConfig.Username, authConfig.Password)

		consumerId, err := oci.GetConsumerIdForRef(registry)
		if err != nil {
			panic(errors.Wrapf(err, "invalid consumer"))
		}
		ocmCtx.CredentialsContext().SetCredentialsForConsumer(consumerId, creds)
	}
	return nil
}

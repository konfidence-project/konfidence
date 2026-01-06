package ocm

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/docker/cli/cli/config/configfile"
	"github.com/konfidence-project/pkg/sanitize"
	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"ocm.software/ocm/api/oci"
	"ocm.software/ocm/api/ocm"
	"ocm.software/ocm/api/tech/oci/identity"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	DefaultConfigMapName      = "vector-deployment-controller-configuration"
	AuthConfigMapKey          = "authenticationSecretRefs"
	KonfidenceSystemNamespace = "konfidence-system"
)

// ContextProvider interface to create an OCM Context
type ContextProvider interface {
	GetOCMContext(ctx context.Context, namespace string, registryUrl string) (ocm.Context, error)
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
func (p *Provider) GetOCMContext(ctx context.Context, namespace string, registryUrl string) (ocm.Context, error) {
	ocmCtx := ocm.DefaultContext()
	secret, err := p.GetCredentials(ctx, namespace, registryUrl)
	if err != nil {
		return nil, err
	}

	// no credentials available, use default ocm context
	if secret == nil {
		return ocmCtx, nil
	}

	err = parseDockerConfigJsonSecret(secret, ocmCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse secret %q: %w", secret.Name, err)
	}

	return ocmCtx, nil
}

func (p *Provider) GetCredentials(ctx context.Context, namespace string, registryUrl string) (*v1.Secret, error) {
	log := logf.FromContext(ctx)

	// TODO this might not be a plain URL. Check again/possible refactor code
	// TODO when OCM version 2 has been released
	domain, err := extractDomain(registryUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to extract domain from registry url: %w", err)
	}

	if domain == "" {
		log.Info(fmt.Sprintf("Could not extract domain from url %q", registryUrl))
		return nil, nil
	}

	// first try to get via default configmap
	secretNameByConfigMap, err := p.getSecretByConfigmap(ctx, domain)
	if err != nil {
		return nil, err
	}

	secretName := secretNameByConfigMap
	if secretName == "" {
		// alternatively try to get secret by domain name
		secretName = sanitize.DNSSubdomainName(domain)
	}

	secret := &v1.Secret{}
	err = p.kubeClient.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      secretName,
	}, secret)

	if err != nil {
		// if the config map does contain a secret reference but the secret
		// does not exist treat this as an error
		if secretNameByConfigMap != "" {
			return nil, err
		}
		return nil, client.IgnoreNotFound(err)
	}

	if secret.Type != v1.SecretTypeDockerConfigJson {
		log.Info(fmt.Sprintf("Secret %q has unsupported type %q and will be ignored", secret.Name, secret.Type))
		return nil, nil
	}

	return secret, nil
}

func parseDockerConfigJsonSecret(s *v1.Secret, ocmCtx ocm.Context) error {
	dockerConfigJson, ok := s.Data[v1.DockerConfigJsonKey]
	if !ok {
		return fmt.Errorf("secret %q does not contain key %q", s.Name, v1.DockerConfigJsonKey)
	}

	var dockerConfig configfile.ConfigFile
	if err := json.Unmarshal(dockerConfigJson, &dockerConfig); err != nil {
		return fmt.Errorf("failed to unmarshal docker config json from secret %q: %w", s.Name, err)
	}

	for registry, authConfig := range dockerConfig.AuthConfigs {
		consumerId, err := oci.GetConsumerIdForRef(registry)
		if err != nil {
			return fmt.Errorf("invalid consumer %w", err)
		}

		creds := identity.SimpleCredentials(authConfig.Username, authConfig.Password)
		ocmCtx.CredentialsContext().SetCredentialsForConsumer(consumerId, creds)
	}

	return nil
}

func (p *Provider) getSecretByConfigmap(ctx context.Context, domainName string) (string, error) {
	log := logf.FromContext(ctx)
	configMap := &v1.ConfigMap{}
	// config map must be in konfidence system namespace
	err := p.kubeClient.Get(ctx, types.NamespacedName{
		Namespace: KonfidenceSystemNamespace,
		Name:      DefaultConfigMapName,
	}, configMap)

	if err != nil && !errors.IsNotFound(err) {
		return "", err
	}
	if err != nil && errors.IsNotFound(err) {
		return "", nil
	}

	authConfig, ok := configMap.Data[AuthConfigMapKey]
	if !ok {
		log.Info("Could not find any data in ConfigMap with AuthConfigMapKey", "key", AuthConfigMapKey)
		return "", nil
	}

	authMap := make(map[string]string)
	err = yaml.Unmarshal([]byte(authConfig), authMap)
	if err != nil {
		log.Info("Error unmarshalling authConfig")
		return "", nil
	}

	secret, ok := authMap[domainName]
	if !ok {
		log.Info("Could not find a map entry for domain", "domainName", domainName)
		return "", nil
	}

	return secret, nil
}

func extractDomain(rawURL string) (string, error) {
	if !strings.Contains(rawURL, "://") {
		rawURL = "//" + rawURL
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	domain := u.Hostname()
	domain = strings.TrimPrefix(domain, "www.")
	return domain, nil
}

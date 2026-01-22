package ocm

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"ocm.software/open-component-model/bindings/go/oci"
	urlresolver "ocm.software/open-component-model/bindings/go/oci/resolver/url"
	"ocm.software/open-component-model/bindings/go/repository"
	"ocm.software/open-component-model/bindings/go/runtime"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

type Client interface {
	GetLatestComponentVersion(context.Context, string) (string, error)
}

type OCIClient struct {
}

func (OCIClient) GetLatestComponentVersion(ctx context.Context, registryAndComponent string) (string, error) {
	repo, err := getOCMRepository(registryAndComponent)
	if err != nil {
		return "", fmt.Errorf("unable to get ocm component version repository for %s: %w", registryAndComponent, err)
	}

	component, err := getOCMComponent(registryAndComponent)
	if err != nil {
		return "", fmt.Errorf("unable to get ocm component from %s: %w", registryAndComponent, err)
	}

	componentVersions, err := repo.ListComponentVersions(ctx, component)
	if err != nil && strings.Contains(err.Error(), "repository name not known to registry") {
		return "", fmt.Errorf("no versions found for component %s: %w", component, err)
	} else if err != nil {
		return "", fmt.Errorf("unable to list component versions for %s: %w", component, err)
	}

	if len(componentVersions) == 0 {
		return "", fmt.Errorf("no versions found for component %s: %w", component, err)
	}

	return componentVersions[0], nil
}

func getOCMRepository(registryAndComponent string) (repository.ComponentVersionRepository, error) {
	registryUrl, err := getRegistryURL(registryAndComponent)
	if err != nil {
		return nil, err
	}
	registry := registryUrl.Host + registryUrl.Path

	// todo: quick and dirty auth client for local testing only!!
	authClient := &auth.Client{
		Client: retry.DefaultClient,
		Header: map[string][]string{"User-Agent": {"gcp-stage-configuration-controller"}},
		Credential: auth.StaticCredential("localhost:5100", auth.Credential{
			Username: "",
			Password: "",
		}),
	}

	resolver, err := urlresolver.New(
		urlresolver.WithBaseURL(registry),
		urlresolver.WithPlainHTTP(true),
		urlresolver.WithBaseClient(authClient),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create url resolver for registry: %s: %w", registry, err)
	}
	repo, err := oci.NewRepository(oci.WithResolver(resolver))
	if err != nil {
		return nil, fmt.Errorf("unable to create oci repository for registry: %s: %w", registry, err)
	}
	return repo, nil
}

func getRegistryURL(registryAndComponent string) (*url.URL, error) {
	data := strings.Split(registryAndComponent, "//")
	var (
		err         error
		registryUrl *url.URL
	)
	switch len(data) {
	case 3:
		registryStr := data[0] + "//" + data[1]
		if registryUrl, err = url.Parse(registryStr); err != nil {
			return nil, fmt.Errorf("unable parse *url.URL from identified oci url.URL{}: %s: %w", registryStr, err)
		}
	case 2:
		if registryUrl, err = runtime.ParseURLAndAllowNoScheme(data[0]); err != nil {
			return nil, fmt.Errorf("unable parse *url.URL from identified oci url.URL{}: %s: %w", data[0], err)
		}
	default:
		return nil, fmt.Errorf("unable to parse oci url.URL{} from registryAndOCMComponent input: %s. "+
			"Format is not supported", registryAndComponent)
	}
	return registryUrl, nil
}

func getOCMComponent(registryAndComponent string) (string, error) {
	registryUrl, err := getRegistryURL(registryAndComponent)
	if err != nil {
		return "", fmt.Errorf("unable to cut registry url from input (%s) to extract ocm component. "+
			"Failed to extract registry url: %w",
			registryAndComponent, err)
	}
	registry := strings.TrimPrefix(registryUrl.String(), "//")
	return strings.TrimPrefix(registryAndComponent, registry+"//"), nil
}

package ocm

import (
	"context"
	"fmt"

	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"

	descriptor "ocm.software/open-component-model/bindings/go/descriptor/runtime"
	"ocm.software/open-component-model/bindings/go/oci"
	ocicredentials "ocm.software/open-component-model/bindings/go/oci/credentials"
	"ocm.software/open-component-model/bindings/go/oci/looseref"
	urlresolver "ocm.software/open-component-model/bindings/go/oci/resolver/url"
	ociaccess "ocm.software/open-component-model/bindings/go/oci/spec/access"
	v1 "ocm.software/open-component-model/bindings/go/oci/spec/access/v1"
	ocicredsv1 "ocm.software/open-component-model/bindings/go/oci/spec/credentials/v1"
	credidentityv1 "ocm.software/open-component-model/bindings/go/oci/spec/identity/v1"
	ociv1 "ocm.software/open-component-model/bindings/go/oci/spec/repository/v1/oci"
	"ocm.software/open-component-model/bindings/go/runtime"
)

type plainHTTPResourceRepository struct {
	plainHTTP bool
}

func newPlainHTTPResourceRepository(plainHTTP bool) *plainHTTPResourceRepository {
	return &plainHTTPResourceRepository{plainHTTP: plainHTTP}
}

func (p *plainHTTPResourceRepository) GetResourceRepositoryScheme() *runtime.Scheme {
	return ociaccess.Scheme
}

func (p *plainHTTPResourceRepository) GetResourceDigestProcessorCredentialConsumerIdentity(
	_ context.Context, resource *descriptor.Resource) (runtime.Identity, error) {
	t := resource.Access.GetType()
	obj, err := p.GetResourceRepositoryScheme().NewObject(t)
	if err != nil {
		return nil, fmt.Errorf("error creating new object for type %s: %w", t, err)
	}
	if err := p.GetResourceRepositoryScheme().Convert(resource.Access, obj); err != nil {
		return nil, fmt.Errorf("error converting access to object of type %s: %w", t, err)
	}
	access, ok := obj.(*v1.OCIImage)
	if !ok {
		return nil, fmt.Errorf("unsupported type %s for getting identity", obj.GetType())
	}
	ref, err := looseref.ParseReference(access.ImageReference)
	if err != nil {
		return nil, fmt.Errorf("error parsing image reference %q: %w", access.ImageReference, err)
	}
	identity, err := runtime.ParseURLToIdentity(ref.RegistryWithScheme())
	if err != nil {
		return nil, fmt.Errorf("error parsing URL to identity: %w", err)
	}
	identity.SetType(credidentityv1.Type)
	return identity, nil
}

func (p *plainHTTPResourceRepository) ProcessResourceDigest(
	ctx context.Context, resource *descriptor.Resource, credentials runtime.Typed) (*descriptor.Resource, error) {
	repo, err := p.resolveOCIImageRepo(resource, credentials)
	if err != nil {
		return nil, err
	}
	resource = resource.DeepCopy()
	t := resource.Access.GetType()
	obj, err := p.GetResourceRepositoryScheme().NewObject(t)
	if err != nil {
		return nil, fmt.Errorf("error creating new object for type %s: %w", t, err)
	}
	if err := p.GetResourceRepositoryScheme().Convert(resource.Access, obj); err != nil {
		return nil, fmt.Errorf("error converting access to object of type %s: %w", t, err)
	}
	resource.Access = obj
	resource, err = repo.ProcessResourceDigest(ctx, resource)
	if err != nil {
		return nil, fmt.Errorf("error processing resource digest: %w", err)
	}
	return resource, nil
}

// resolveOCIImageRepo is copied from OCM's unexported method:
// open-component-model/open-component-model/blob/main/bindings/go/oci/repository/resource/resource_repository
func (p *plainHTTPResourceRepository) resolveOCIImageRepo(
	resource *descriptor.Resource, credentials runtime.Typed) (*oci.Repository, error) {
	t := resource.Access.GetType()
	obj, err := p.GetResourceRepositoryScheme().NewObject(t)
	if err != nil {
		return nil, fmt.Errorf("error creating new object for type %s: %w", t, err)
	}
	if err := p.GetResourceRepositoryScheme().Convert(resource.Access, obj); err != nil {
		return nil, fmt.Errorf("error converting access to object of type %s: %w", t, err)
	}
	access, ok := obj.(*v1.OCIImage)
	if !ok {
		return nil, fmt.Errorf("unsupported access type %s: expected OCI image", t)
	}
	ref, err := looseref.ParseReference(access.ImageReference)
	if err != nil {
		return nil, fmt.Errorf("error parsing image reference %q: %w", access.ImageReference, err)
	}
	baseURL := ref.RegistryWithScheme()

	var ociCredentials *ocicredsv1.OCICredentials
	if credentials != nil {
		ociCredentials, err = ocicredsv1.ConvertToOCICredentials(credentials)
		if err != nil {
			return nil, fmt.Errorf("error converting credentials: %w", err)
		}
	}
	return p.buildRepository(&ociv1.Repository{BaseUrl: baseURL}, ociCredentials)
}

// buildRepository is createRepository from OCM with WithPlainHTTP added
func (p *plainHTTPResourceRepository) buildRepository(
	spec *ociv1.Repository, credentials *ocicredsv1.OCICredentials) (*oci.Repository, error) {
	url, err := runtime.ParseURLAndAllowNoScheme(spec.BaseUrl)
	if err != nil {
		return nil, fmt.Errorf("invalid URL %q: %w", spec.BaseUrl, err)
	}
	urlString := url.Host + url.Path

	resolver, err := urlresolver.New(
		urlresolver.WithBaseURL(urlString),
		urlresolver.WithPlainHTTP(p.plainHTTP),
		urlresolver.WithBaseClient(&auth.Client{
			Client:     retry.DefaultClient,
			Credential: auth.StaticCredential(url.Host, ocicredentials.MapCredentials(credentials)),
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create URL resolver: %w", err)
	}
	return oci.NewRepository(oci.WithResolver(resolver))
}

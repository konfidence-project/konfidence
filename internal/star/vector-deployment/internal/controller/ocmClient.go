package controller

import (
	"context"
	"github.com/docker/cli/cli/config/configfile"
	utilErrors "github.com/mandelsoft/goutils/errors"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"ocm.software/ocm/api/oci"
	"ocm.software/ocm/api/ocm"
	"ocm.software/ocm/api/ocm/extensions/repositories/ocireg"
	"ocm.software/ocm/api/tech/oci/identity"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const konfidenceNamespace = "konfidence-system"

func initOcmContext(kubeClient client.Client, ctx context.Context) (*ocm.Context, error) {
	ocmCtx := ocm.DefaultContext()

	secretList := v1.SecretList{}
	if err := kubeClient.List(ctx, &secretList, client.InNamespace(konfidenceNamespace)); err != nil {
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
	return &ocmCtx, nil
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

func fetchOcm(ref ocm.RefSpec, ctx ocm.Context) (*ocm.ComponentVersionAccess, error) {
	repoHost := ref.UniformRepositorySpec.Host
	consumerId, err := oci.GetConsumerIdForRef(repoHost)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get consumer id for host %q", repoHost)
	}

	regCreds, err := ctx.CredentialsContext().GetCredentialsForConsumer(consumerId)
	if err != nil && !utilErrors.IsErrUnknownKind(err, "consumer") {
		return nil, errors.Wrapf(err, "failed to get credentials for consumer %q", consumerId.String())
	}

	spec := ocireg.NewRepositorySpec(ref.UniformRepositorySpec.String())
	var repo ocm.Repository
	if regCreds != nil {
		repo, err = ctx.RepositoryForSpec(spec, regCreds)
	} else {
		repo, err = ctx.RepositoryForSpec(spec)
	}
	if err != nil {
		return nil, errors.Wrapf(err, "cannot setup repository")
	}
	defer repo.Close()

	cv, err := fetchOcmComponentVersionFromRepo(repo, ref.Component, *ref.Version)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to fetch component version %q from repository %q", ref.Component, ref.UniformRepositorySpec.String())
	}

	return cv, nil
}

func parseComponentVersionUrl(ref string) (ocm.RefSpec, error) {
	ocmRef, err := ocm.ParseRef(ref)
	if err != nil {
		return ocm.RefSpec{}, errors.Wrapf(err, "invalid vector reference %q", ref)
	}
	if !ocmRef.IsVersion() {
		return ocm.RefSpec{}, errors.Errorf("vector reference %q is not a version", ref)
	}
	return ocmRef, nil
}

func fetchOcmComponentVersionFromRepo(repo ocm.Repository, component string, version string) (*ocm.ComponentVersionAccess, error) {
	cv, err := repo.LookupComponentVersion(component, version)
	if err != nil {
		return nil, errors.Wrapf(err, "cannot lookup component version")
	}

	return &cv, nil
}

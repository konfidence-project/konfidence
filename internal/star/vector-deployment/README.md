[![REUSE status](https://api.reuse.software/badge/github.com/konfidence-project/landscape-vector-deployment-controller)](https://api.reuse.software/info/github.com/konfidence-project/landscape-vector-deployment-controller)

# landscape-vector-deployment-controller

## About this project

The Deployment Controller is responsible for managing the lifecycle of VectorDeployments, including downloading the vector image from the OCI repository and creating the related ArtifactDeployments and VectorAssignments.

## Requirements and Setup

### OCM Authentication Setup

When accessing OCM artifacts the Deployment Controller needs (optional) authentication credentials to access one or more (remote) OCI registries.
First the controller extracts the domain name from the OCI registry url and then tries to lookup a Secret reference by domain name in a controller specific ConfigMap.
If a matching reference is found the credentials are extracted from the referenced Secret. If the ConfigMap does not exist or no matching entry has been found
the controller tries to directly get the Secret by domain name. If this Secret also not exists then no OCI credentials are configured.

So there are two options to configure the OCI registry credentials:
1. A Deployment Controller specific ConfigMap `vector-deployment-controller-configuration` in the `konfidence-system` namespace that contains one or more Secret references by domain name.
2. A Secret with the domain as name that exists in the namespace of the vector deployment resource

> **Note:**
> At the moment the secret has to be of type `kubernetes.io/dockerconfigjson`

#### Example with config map
1. Create a Secret with the OCI credentials of type `kubernetes.io/dockerconfigjson`. The Secret must be in the same namespace as the vector deployment resource.
```yaml
apiVersion: v1
data:
  .dockerconfigjson: eyJhdXRocyI6eyJyZWdpc3RyeS5leGFtcGxlLmNvbSI6eyJ1c2VybmFtZSI6InRlc3QiLCJwYXNzd29yZCI6InRlc3QiLCJlbWFpbCI6InRlc3RAc2FwLmNvbSIsImF1dGgiOiJkR1Z6ZERwMFpYTjAifX19
kind: Secret
metadata:
  creationTimestamp: "2026-01-05T11:36:58Z"
  name: my-secret-123
  namespace: default
  resourceVersion: "1766473"
  uid: c17e1c74-e10b-4d1f-99a3-0a3da7f4b5a8
type: kubernetes.io/dockerconfigjson
```

2. Create the ConfigMap referencing the Secret by domain name. The ConfigMap must be in the `konfidence-system` namespace.
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: vector-deployment-controller-configuration
  namespace: konfidence-system
data:
  authenticationSecretRefs: |
    registry.example.com: my-secret-123
```

### Setup Git hooks

We use git hooks to check the conventional-commit formatting at "commit-msg".

```bash
make install-git-hooks    # install all git hooks with prek
make uninstall-git-hooks  # uninstall all git hooks with prek
```

## Support, Feedback, Contributing

This project is open to feature requests/suggestions, bug reports etc. via [GitHub issues](https://github.com/konfidence-project/<your-project>/issues). Contribution and feedback are encouraged and always welcome. For more information about how to contribute, the project structure, as well as additional contribution information, see our [Contribution Guidelines](CONTRIBUTING.md).

## Security / Disclosure
If you find any bug that may be a security problem, please follow our instructions at [in our security policy](https://github.com/konfidence-project/<your-project>/security/policy) on how to report it. Please do not create GitHub issues for security-related doubts or problems.

## Code of Conduct

We as members, contributors, and leaders pledge to make participation in our community a harassment-free experience for everyone. By participating in this project, you agree to abide by its [Code of Conduct](https://github.com/konfidence-project/.github/blob/main/CODE_OF_CONDUCT.md) at all times.

## Licensing

Copyright 2025 SAP SE or an SAP affiliate company and landscape-vector-deployment-controller contributors. Please see our [LICENSE](LICENSE) for copyright and license information. Detailed information including third-party components and their licensing/copyright information is available [via the REUSE tool](https://api.reuse.software/info/github.com/konfidence-project/<your-project>).

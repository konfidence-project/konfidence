# konfidence

![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square)

Konfidence operator for orchestrating multi-service deployments (with bundled CRDs).

**Homepage:** <https://github.com/konfidence-project/konfidence>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| Konfidence maintainers |  |  |

## Source Code

* <https://github.com/konfidence-project/konfidence>

## Requirements

Kubernetes: `>=1.27.0-0`

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | object | `{}` | Affinity rules for the controller Pod. |
| api.database.maxConnIdleTime | string | `"5m"` | Maximum idle time of a connection. |
| api.database.maxConnLifetime | string | `"30m"` | Maximum lifetime of a connection. |
| api.database.maxConns | int | `10` | Maximum number of open connections. |
| api.database.minConns | int | `5` | Minimum number of idle connections kept open. |
| api.enabled | bool | `true` | Deploy the API server. When false, all API server resources (Deployment, Service, RBAC) are skipped. |
| api.env | list | `[]` | Additional environment variables for the API server container. |
| api.extraArgs | list | `[]` | Extra CLI arguments appended to the API server container args list. |
| api.hostAliases | list | `[]` | Host aliases added to the API pod. Used by local development to reach an identity provider on the host. |
| api.image.pullPolicy | string | `"IfNotPresent"` | API server image pull policy. |
| api.image.repository | string | `"ghcr.io/konfidence-project/api"` | API server image repository. |
| api.image.tag | string | `""` | API server image tag. Defaults to `.Chart.AppVersion` when empty. |
| api.ingress.annotations | object | `{}` | Annotations to add to the Ingress, for example for cert-manager. |
| api.ingress.className | string | `""` | IngressClass to use. |
| api.ingress.enabled | bool | `false` | Create an Ingress for the API server. |
| api.ingress.hosts | list | `[{"host":"","paths":[{"path":"/","pathType":"Prefix"}]}]` | Hosts and paths routed to the API server. `host` is required on each entry. |
| api.ingress.tls | list | `[]` | TLS configuration of the Ingress. |
| api.oidc.allowReturnUrls | list | `[]` | URLs the login flow may redirect back to after authentication. |
| api.oidc.authorizationURL | string | `""` | Authorization endpoint. Defaults to the discovery document when empty. |
| api.oidc.clientId | string | `""` | OAuth client id registered at the provider. |
| api.oidc.clientSecretRef.key | string | `"client-secret"` | Key inside that Secret holding the client secret. |
| api.oidc.clientSecretRef.name | string | `""` | Name of the Secret in the release namespace holding the client secret. |
| api.oidc.deviceAuthURL | string | `""` | Device authorization endpoint. Defaults to the discovery document when empty. |
| api.oidc.enabled | bool | `true` | Enable login through an OpenID Connect provider. When true, `issuerURL` must be set or the API server refuses to start. |
| api.oidc.issuerURL | string | `""` | Issuer URL of the OpenID Connect provider. |
| api.oidc.jwksCacheTTL | string | `"15m"` | How long fetched JWKS signing keys are cached before they are fetched again. |
| api.oidc.jwksURL | string | `""` | JWKS endpoint. Defaults to the discovery document when empty. |
| api.oidc.pkceEnabled | bool | `true` | Use PKCE for the authorization code flow. |
| api.oidc.redirectURL | string | `""` | Redirect URL registered at the provider, ending in `/api/v1/auth/callback`. |
| api.oidc.scopes | string | `"openid,profile,email"` | Comma-separated scopes requested at login. Add the scope that makes your provider include group membership, since role bindings match users by group. |
| api.oidc.stateExpiration | string | `"15m"` | Lifetime of the login state parameter. |
| api.oidc.tokenURL | string | `""` | Token endpoint. Defaults to the value from the provider's discovery document when empty. |
| api.oidc.userInfoURL | string | `""` | UserInfo endpoint. Defaults to the discovery document when empty. |
| api.podAnnotations | object | `{}` | Annotations to add to the API server Pod. |
| api.podDisruptionBudget.enabled | bool | `false` | Enable the PodDisruptionBudget for the API server. |
| api.podDisruptionBudget.maxUnavailable | int | `1` | Maximum number of pods that may be unavailable. |
| api.podDisruptionBudget.minAvailable | string | `nil` | Minimum number of pods that must be available. Set exactly one of `minAvailable` or `maxUnavailable`. |
| api.podLabels | object | `{}` | Labels to add to the API server Pod. |
| api.replicas | int | `1` | Number of API server replicas. |
| api.resources | object | `{}` | Resource requests and limits for the API server container. |
| api.server.addr | string | `":8090"` | TCP address the API server listens on inside the container. |
| api.server.logLevel | string | `"info"` | Log level of the API server. |
| api.server.readTimeout | string | `"10s"` | HTTP read timeout. |
| api.server.shutdownTimeout | string | `"15s"` | Grace period for in-flight requests on shutdown. |
| api.server.writeTimeout | string | `"10s"` | HTTP write timeout. |
| api.service.annotations | object | `{}` | Annotations to add to the Service. |
| api.service.nodePort | string | `""` | Node port when `type` is `NodePort`. |
| api.service.port | int | `8090` | Service port. |
| api.service.type | string | `"ClusterIP"` | Service type. `ClusterIP` is reachable inside the cluster only. |
| api.session.cleanupInterval | string | `"15m"` | Interval at which expired sessions are removed. |
| api.session.cookie.httpOnly | bool | `true` | Mark the session cookie HttpOnly. |
| api.session.cookie.name | string | `"kden-session"` | Name of the session cookie. |
| api.session.cookie.sameSite | string | `"SameSiteStrictMode"` | SameSite attribute of the session cookie. |
| api.session.cookie.secure | bool | `true` | Mark the session cookie Secure. Browsers then send it over HTTPS only. |
| api.session.expiration | string | `"12h"` | Session lifetime. |
| api.session.storageType | string | `"in-memory"` | Session storage backend. |
| api.volumeMounts | list | `[]` | Extra volume mounts on the API server container. |
| api.volumes | list | `[]` | Extra volumes mounted onto the API server Pod. |
| containerSecurityContext | object | `{"allowPrivilegeEscalation":false,"capabilities":{"drop":["ALL"]},"readOnlyRootFilesystem":true}` | Container-level security context. |
| controller.controllers | string | `"*"` | Comma-separated list of sub-controllers, or `"*"` for all. |
| controller.healthProbeBindAddress | string | `":8081"` | Bind address for the health probe endpoint. |
| controller.install | bool | `true` | Deploy the controller. When false, the chart renders only the CRDs (gated by `crd.install`) and skips Deployment, ServiceAccount, RBAC, Service, ServiceMonitor, and PodDisruptionBudget. Useful for installing CRDs into a local development cluster without running the controller. |
| controller.leaderElection | bool | `true` | Enable leader election for the controller. |
| controller.leaseId | string | `"konfidence-operator.konfidence.cloud"` | Lease ID used for leader election. |
| controller.metricsBindAddress | string | `":8080"` | Bind address for the Prometheus `/metrics` endpoint. Set to `"0"` to disable the metrics server entirely (also suppresses the metrics container port, Service, and ServiceMonitor). |
| crd.annotations | object | `{}` | Extra annotations merged onto every CRD. Useful for tools that inspect CRD metadata (e.g. Argo CD: `argocd.argoproj.io/sync-options: ServerSideApply=true` to avoid the client-side apply size limit). |
| crd.install | bool | `true` | Apply CRDs alongside the controller. Disable for GitOps setups that manage CRDs out-of-band. |
| crd.keep | bool | `true` | On `helm uninstall`, retain CRDs to protect existing CRs. Strongly recommended. |
| crd.labels | object | `{}` | Extra labels merged onto every CRD. |
| env | list | `[]` | Additional environment variables for the controller container. |
| extraArgs | list | `[]` | Additional command-line arguments passed to the controller binary. |
| fullnameOverride | string | `""` | Override the fully-qualified app name used in resource names. |
| image.pullPolicy | string | `"IfNotPresent"` | Operator image pull policy. |
| image.repository | string | `"ghcr.io/konfidence-project/konfidence-operator"` | Operator image repository. |
| image.tag | string | `""` | Operator image tag. Defaults to `.Chart.AppVersion` when empty. |
| imagePullSecrets | list | `[]` | Image pull secrets for private registries. |
| nameOverride | string | `""` | Override the chart name used in resource names. |
| nodeSelector | object | `{}` | Node selector for the controller Pod. |
| podAnnotations | object | `{}` | Annotations to add to the controller Pod. |
| podDisruptionBudget.enabled | bool | `false` | Enable the PodDisruptionBudget for the controller. |
| podDisruptionBudget.maxUnavailable | int | `1` | Maximum number of pods that may be unavailable. |
| podDisruptionBudget.minAvailable | string | `nil` | Minimum number of pods that must be available. Set exactly one of `minAvailable` or `maxUnavailable`. Both accept an integer or a percentage string (e.g. `"50%"`). |
| podLabels | object | `{}` | Labels to add to the controller Pod. |
| replicas | int | `1` | Number of controller replicas. Use `>= 2` with `controller.leaderElection: true` for HA. |
| resources | object | `{"limits":{"cpu":"500m","memory":"512Mi"},"requests":{"cpu":"100m","memory":"128Mi"}}` | Resource requests and limits for the controller container. |
| securityContext | object | `{"runAsNonRoot":true,"runAsUser":65532,"seccompProfile":{"type":"RuntimeDefault"}}` | Pod-level security context. |
| serviceAccount.annotations | object | `{}` | Annotations to add to the ServiceAccount. |
| serviceAccount.create | bool | `true` | Create a ServiceAccount for the controller. |
| serviceAccount.name | string | `""` | Name of the ServiceAccount to use. Defaults to the fully-qualified app name when empty. |
| serviceMonitor.enabled | bool | `false` | Enable the ServiceMonitor resource. Requires the `monitoring.coreos.com` CRDs. |
| serviceMonitor.interval | string | `"30s"` | Scrape interval. |
| serviceMonitor.labels | object | `{}` | Extra labels merged onto the ServiceMonitor — typically the label selector your Prometheus instance uses (e.g. `release: kube-prometheus`). |
| serviceMonitor.metricRelabelings | list | `[]` | Metric relabeling rules applied after scraping. |
| serviceMonitor.namespace | string | `""` | Namespace to create the ServiceMonitor in. Defaults to the release namespace. Override when your Prometheus instance only watches a specific namespace. |
| serviceMonitor.relabelings | list | `[]` | Relabeling rules applied to scraped samples before ingestion. |
| serviceMonitor.scrapeTimeout | string | `"10s"` | Scrape timeout. |
| tolerations | list | `[]` | Tolerations for the controller Pod. |
| volumeMounts | list | `[]` | Extra volume mounts on the manager container, corresponding to `volumes`. |
| volumes | list | `[]` | Extra volumes mounted onto the manager Pod. Use these for additional secrets, CA bundles, or other files the controller needs at runtime. An emptyDir is always mounted at `/tmp` (the OCM transfer writes there, and the root filesystem is read-only). |
| webhook.annotations | object | `{}` | Extra annotations for the ValidatingWebhookConfiguration resource, for example `cert-manager.io/inject-ca-from`. |
| webhook.caBundle | string | `""` | Base64-encoded CA certificate for the webhook's TLS certificate. Leave empty when using a CA injection mechanism (e.g. cert-manager). REQUIRED for self-signed certificates without an injection mechanism. WARNING: if empty and no injection occurs, the API server skips TLS verification (not recommended). |
| webhook.certDir | string | `"/tmp/k8s-webhook-server/serving-certs"` | Directory inside the container where TLS certificates are mounted. The Secret specified by `certificateSecret` will be mounted here. |
| webhook.certificateSecret | string | `"konfidence-webhook-server-cert"` | Name of the Secret containing `tls.crt` and `tls.key` in the release namespace. You must create this Secret before installing the chart, or set `enabled: false`. |
| webhook.enabled | bool | `true` | Enable the validating admission webhooks. Requires the TLS Secret named by `webhook.certificateSecret` to exist before the controller starts. |
| webhook.failurePolicy | string | `"Fail"` | `Fail` blocks requests if the webhook is unavailable. Set to `Ignore` for less strict validation (allow requests if the webhook is down). |
| webhook.labels | object | `{}` | Extra labels for the ValidatingWebhookConfiguration resource. |
| webhook.port | int | `9443` | Port the webhook server listens on inside the container. |

----------------------------------------------
Autogenerated from chart metadata using [helm-docs v1.14.2](https://github.com/norwoodj/helm-docs/releases/v1.14.2)

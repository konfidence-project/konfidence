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
| affinity | object | `{}` |  |
| api.database.maxConnIdleTime | string | `"5m"` |  |
| api.database.maxConnLifetime | string | `"30m"` |  |
| api.database.maxConns | int | `10` |  |
| api.database.minConns | int | `5` |  |
| api.enabled | bool | `true` |  |
| api.env | list | `[]` |  |
| api.extraArgs | list | `[]` |  |
| api.image.pullPolicy | string | `"IfNotPresent"` |  |
| api.image.repository | string | `"ghcr.io/konfidence-project/api"` |  |
| api.image.tag | string | `""` |  |
| api.ingress.annotations | object | `{}` |  |
| api.ingress.className | string | `""` |  |
| api.ingress.enabled | bool | `false` |  |
| api.ingress.hosts[0].host | string | `""` |  |
| api.ingress.hosts[0].paths[0].path | string | `"/"` |  |
| api.ingress.hosts[0].paths[0].pathType | string | `"Prefix"` |  |
| api.ingress.tls | list | `[]` |  |
| api.oidc.allowReturnUrls | list | `[]` |  |
| api.oidc.authorizationURL | string | `""` |  |
| api.oidc.clientId | string | `""` |  |
| api.oidc.clientSecretRef.key | string | `"client-secret"` |  |
| api.oidc.clientSecretRef.name | string | `""` |  |
| api.oidc.deviceAuthURL | string | `""` |  |
| api.oidc.enabled | bool | `true` |  |
| api.oidc.issuerURL | string | `""` |  |
| api.oidc.jwksURL | string | `""` |  |
| api.oidc.pkceEnabled | bool | `true` |  |
| api.oidc.redirectURL | string | `""` |  |
| api.oidc.scopes | string | `"openid,profile,email"` |  |
| api.oidc.stateExpiration | string | `"15m"` |  |
| api.oidc.tokenURL | string | `""` |  |
| api.oidc.userInfoURL | string | `""` |  |
| api.podAnnotations | object | `{}` |  |
| api.podDisruptionBudget.enabled | bool | `false` |  |
| api.podDisruptionBudget.maxUnavailable | int | `1` |  |
| api.podDisruptionBudget.minAvailable | string | `nil` |  |
| api.podLabels | object | `{}` |  |
| api.replicas | int | `1` |  |
| api.resources | object | `{}` |  |
| api.server.addr | string | `":8090"` |  |
| api.server.logLevel | string | `"info"` |  |
| api.server.readTimeout | string | `"10s"` |  |
| api.server.shutdownTimeout | string | `"15s"` |  |
| api.server.writeTimeout | string | `"10s"` |  |
| api.service.annotations | object | `{}` |  |
| api.service.nodePort | string | `""` |  |
| api.service.port | int | `8090` |  |
| api.service.type | string | `"ClusterIP"` |  |
| api.session.cleanupInterval | string | `"15m"` |  |
| api.session.cookie.httpOnly | bool | `true` |  |
| api.session.cookie.name | string | `"kden-session"` |  |
| api.session.cookie.sameSite | string | `"SameSiteStrictMode"` |  |
| api.session.cookie.secure | bool | `true` |  |
| api.session.expiration | string | `"12h"` |  |
| api.session.storageType | string | `"in-memory"` |  |
| api.volumeMounts | list | `[]` |  |
| api.volumes | list | `[]` |  |
| containerSecurityContext.allowPrivilegeEscalation | bool | `false` |  |
| containerSecurityContext.capabilities.drop[0] | string | `"ALL"` |  |
| containerSecurityContext.readOnlyRootFilesystem | bool | `true` |  |
| controller.controllers | string | `"*"` |  |
| controller.healthProbeBindAddress | string | `":8081"` |  |
| controller.install | bool | `true` |  |
| controller.leaderElection | bool | `true` |  |
| controller.leaseId | string | `"konfidence-operator.konfidence.cloud"` |  |
| controller.metricsBindAddress | string | `":8080"` |  |
| crd.annotations | object | `{}` |  |
| crd.install | bool | `true` |  |
| crd.keep | bool | `true` |  |
| crd.labels | object | `{}` |  |
| env | list | `[]` |  |
| extraArgs | list | `[]` |  |
| fullnameOverride | string | `""` |  |
| image.pullPolicy | string | `"IfNotPresent"` |  |
| image.repository | string | `"ghcr.io/konfidence-project/konfidence-operator"` |  |
| image.tag | string | `""` |  |
| imagePullSecrets | list | `[]` |  |
| nameOverride | string | `""` |  |
| nodeSelector | object | `{}` |  |
| podAnnotations | object | `{}` |  |
| podDisruptionBudget.enabled | bool | `false` |  |
| podDisruptionBudget.maxUnavailable | int | `1` |  |
| podDisruptionBudget.minAvailable | string | `nil` |  |
| podLabels | object | `{}` |  |
| replicas | int | `1` |  |
| resources.limits.cpu | string | `"500m"` |  |
| resources.limits.memory | string | `"512Mi"` |  |
| resources.requests.cpu | string | `"100m"` |  |
| resources.requests.memory | string | `"128Mi"` |  |
| securityContext.runAsNonRoot | bool | `true` |  |
| securityContext.runAsUser | int | `65532` |  |
| securityContext.seccompProfile.type | string | `"RuntimeDefault"` |  |
| serviceAccount.annotations | object | `{}` |  |
| serviceAccount.create | bool | `true` |  |
| serviceAccount.name | string | `""` |  |
| serviceMonitor.enabled | bool | `false` |  |
| serviceMonitor.interval | string | `"30s"` |  |
| serviceMonitor.labels | object | `{}` |  |
| serviceMonitor.metricRelabelings | list | `[]` |  |
| serviceMonitor.namespace | string | `""` |  |
| serviceMonitor.relabelings | list | `[]` |  |
| serviceMonitor.scrapeTimeout | string | `"10s"` |  |
| tolerations | list | `[]` |  |
| volumeMounts | list | `[]` |  |
| volumes | list | `[]` |  |
| webhook.annotations | object | `{}` |  |
| webhook.caBundle | string | `""` |  |
| webhook.certDir | string | `"/tmp/k8s-webhook-server/serving-certs"` |  |
| webhook.certificateSecret | string | `"konfidence-webhook-server-cert"` |  |
| webhook.enabled | bool | `true` |  |
| webhook.failurePolicy | string | `"Fail"` |  |
| webhook.labels | object | `{}` |  |
| webhook.port | int | `9443` |  |

----------------------------------------------
Autogenerated from chart metadata using [helm-docs v1.14.2](https://github.com/norwoodj/helm-docs/releases/v1.14.2)

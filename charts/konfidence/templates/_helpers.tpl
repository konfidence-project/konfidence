{{- define "konfidence.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "konfidence.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "konfidence.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "konfidence.labels" -}}
helm.sh/chart: {{ include "konfidence.chart" . }}
app.kubernetes.io/name: {{ include "konfidence.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: konfidence
app.kubernetes.io/component: controller
{{- end -}}

{{- define "konfidence.selectorLabels" -}}
app.kubernetes.io/name: {{ include "konfidence.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: controller
{{- end -}}

{{- define "konfidence.dashboardFullname" -}}
{{- printf "%s-dashboard" (include "konfidence.fullname" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "konfidence.dashboardLabels" -}}
helm.sh/chart: {{ include "konfidence.chart" . }}
app.kubernetes.io/name: {{ include "konfidence.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: konfidence
app.kubernetes.io/component: dashboard
{{- end -}}

{{- define "konfidence.dashboardSelectorLabels" -}}
app.kubernetes.io/name: {{ include "konfidence.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: dashboard
{{- end -}}

{{- define "konfidence.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "konfidence.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{- /*
Extract the port from a `host:port` bind address (e.g. ":8081" -> 8081).
*/ -}}
{{- define "konfidence.bindAddressPort" -}}
{{- regexFind "[0-9]+$" . -}}
{{- end -}}

{{- define "konfidence.crdAnnotations" -}}
{{- if .Values.crd.keep -}}
helm.sh/resource-policy: keep
{{ end -}}
{{- with .Values.crd.annotations }}
{{ toYaml . | trim }}
{{- end }}
{{- end -}}

{{- define "konfidence.crdLabels" -}}
app.kubernetes.io/name: konfidence-crds
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: konfidence
helm.sh/chart: {{ include "konfidence.chart" . }}
{{- with .Values.crd.labels }}
{{ toYaml . }}
{{- end }}
{{- end -}}

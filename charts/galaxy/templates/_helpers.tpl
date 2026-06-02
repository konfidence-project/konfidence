{{- define "galaxy.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "galaxy.fullname" -}}
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

{{- define "galaxy.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "galaxy.labels" -}}
helm.sh/chart: {{ include "galaxy.chart" . }}
app.kubernetes.io/name: {{ include "galaxy.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: konfidence
app.kubernetes.io/component: controller
{{- end -}}

{{- define "galaxy.selectorLabels" -}}
app.kubernetes.io/name: {{ include "galaxy.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: controller
{{- end -}}

{{- define "galaxy.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "galaxy.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{- define "galaxy.crdAnnotations" -}}
{{- if .Values.keepCRDs }}
helm.sh/resource-policy: keep
{{- end }}
{{- end -}}

{{- define "galaxy.crdLabels" -}}
app.kubernetes.io/name: galaxy-crds
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: konfidence
helm.sh/chart: {{ include "galaxy.chart" . }}
{{- end -}}

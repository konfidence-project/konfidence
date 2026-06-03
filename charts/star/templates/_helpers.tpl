{{- define "star.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "star.fullname" -}}
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

{{- define "star.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "star.labels" -}}
helm.sh/chart: {{ include "star.chart" . }}
app.kubernetes.io/name: {{ include "star.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: konfidence
app.kubernetes.io/component: controller
{{- end -}}

{{- define "star.selectorLabels" -}}
app.kubernetes.io/name: {{ include "star.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/component: controller
{{- end -}}

{{- define "star.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "star.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{- define "star.crdAnnotations" -}}
{{- if .Values.crd.keep -}}
helm.sh/resource-policy: keep
{{ end -}}
{{- with .Values.crd.annotations }}
{{ toYaml . | trim }}
{{- end }}
{{- end -}}

{{- define "star.crdLabels" -}}
app.kubernetes.io/name: star-crds
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: konfidence
helm.sh/chart: {{ include "star.chart" . }}
{{- with .Values.crd.labels }}
{{ toYaml . }}
{{- end }}
{{- end -}}

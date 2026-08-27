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

{{- define "konfidence.baseLabels" -}}
helm.sh/chart: {{ include "konfidence.chart" . }}
app.kubernetes.io/name: {{ include "konfidence.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: konfidence
{{- end -}}

{{- define "konfidence.baseSelectorLabels" -}}
app.kubernetes.io/name: {{ include "konfidence.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{- define "konfidence.operatorLabels" -}}
{{ include "konfidence.baseLabels" . }}
app.kubernetes.io/component: operator
{{- end -}}

{{- define "konfidence.operatorSelectorLabels" -}}
{{ include "konfidence.baseSelectorLabels" . }}
app.kubernetes.io/component: operator
{{- end -}}

{{- define "konfidence.apiLabels" -}}
{{ include "konfidence.baseLabels" . }}
app.kubernetes.io/component: api
{{- end -}}

{{- define "konfidence.apiSelectorLabels" -}}
{{ include "konfidence.baseSelectorLabels" . }}
app.kubernetes.io/component: api
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

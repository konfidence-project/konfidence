{{- define "galaxy-crds.annotations" -}}
{{- if .Values.keepOnUninstall }}
helm.sh/resource-policy: keep
{{- end }}
{{- if .Values.useHooks }}
helm.sh/hook: pre-install,pre-upgrade
helm.sh/hook-weight: "-10"
{{- end }}
{{- with .Values.commonAnnotations }}
{{ toYaml . }}
{{- end }}
{{- end -}}

{{- define "galaxy-crds.labels" -}}
app.kubernetes.io/name: galaxy-crds
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: konfidence
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version }}
{{- with .Values.commonLabels }}
{{ toYaml . }}
{{- end }}
{{- end -}}

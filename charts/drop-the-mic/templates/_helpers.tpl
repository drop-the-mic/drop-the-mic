{{- define "dtm.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{- define "dtm.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{- define "dtm.labels" -}}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: drop-the-mic
{{- end }}

{{- define "dtm.operatorLabels" -}}
{{ include "dtm.labels" . }}
app.kubernetes.io/component: operator
app.kubernetes.io/name: dtm-operator
{{- end }}

{{- define "dtm.uiLabels" -}}
{{ include "dtm.labels" . }}
app.kubernetes.io/component: ui
app.kubernetes.io/name: dtm-ui
{{- end }}

{{- define "dtm.authSecretName" -}}
{{- if .Values.ui.auth.existingSecret }}
{{- .Values.ui.auth.existingSecret }}
{{- else }}
{{- printf "%s-auth" (include "dtm.fullname" .) }}
{{- end }}
{{- end }}

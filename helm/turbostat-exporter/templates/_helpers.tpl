{{/*
Chart name, truncated for use in resource names.
*/}}
{{- define "turbostat-exporter.name" -}}
{{- .Chart.Name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Fully qualified app name.
*/}}
{{- define "turbostat-exporter.fullname" -}}
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

{{/*
Common labels.
*/}}
{{- define "turbostat-exporter.labels" -}}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" }}
{{ include "turbostat-exporter.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{/*
Selector labels.
*/}}
{{- define "turbostat-exporter.selectorLabels" -}}
app.kubernetes.io/name: {{ include "turbostat-exporter.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/*
ServiceAccount name.
*/}}
{{- define "turbostat-exporter.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "turbostat-exporter.fullname" .) .Values.serviceAccount.name -}}
{{- else -}}
{{- default "default" .Values.serviceAccount.name -}}
{{- end -}}
{{- end -}}

{{/*
Secret name holding the basic-auth password.
*/}}
{{- define "turbostat-exporter.basicAuthSecretName" -}}
{{- if .Values.basicAuth.existingSecret -}}
{{- .Values.basicAuth.existingSecret -}}
{{- else -}}
{{- include "turbostat-exporter.fullname" . -}}
{{- end -}}
{{- end -}}

{{/*
Secret key holding the basic-auth password.
*/}}
{{- define "turbostat-exporter.basicAuthSecretKey" -}}
{{- if .Values.basicAuth.existingSecret -}}
{{- .Values.basicAuth.existingSecretKey -}}
{{- else -}}
password
{{- end -}}
{{- end -}}

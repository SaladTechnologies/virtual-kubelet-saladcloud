{{- /*
Copyright Salad Technologies, Inc. All Rights Reserved.
SPDX-License-Identifier: APACHE-2.0
*/ -}}

{{/*
Get the chart name and version.
*/}}
{{- define "virtual-kubelet-saladcloud.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end -}}

{{/*
Get the app name (truncated to 63 characters to comply with DNS limitations).
*/}}
{{- define "virtual-kubelet-saladcloud.name" -}}
{{- default "virtual-kubelet-saladcloud" .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end -}}

{{/*
Get the fully qualified app name (truncated to 63 characters to comply with DNS limitations).
*/}}
{{- define "virtual-kubelet-saladcloud.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default "virtual-kubelet-saladcloud" .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end -}}

{{/*
Get the namespace. Enables overriding the namespace for multi-namespace deployments when used as a subchart.
*/}}
{{- define "virtual-kubelet-saladcloud.namespace" -}}
{{- default .Release.Namespace .Values.namespaceOverride | trunc 63 | trimSuffix "-" }}
{{- end -}}

{{/*
Get the labels used by all resources.
*/}}
{{- define "virtual-kubelet-saladcloud.labels" -}}
app.kubernetes.io/component: kubelet
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/version: {{ .Chart.Version | quote }}
helm.sh/chart: {{ include "virtual-kubelet-saladcloud.chart" . }}
{{ include "virtual-kubelet-saladcloud.matchLabels" . }}
{{- with .Values.commonLabels }}
{{ toYaml . }}
{{- end }}
{{- end -}}

{{/*
Get the labels used by the deployment's selector.
*/}}
{{- define "virtual-kubelet-saladcloud.matchLabels" -}}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/name: {{ include "virtual-kubelet-saladcloud.name" . }}
{{- end -}}

{{/*
Get the service account name.
*/}}
{{- define "virtual-kubelet-saladcloud.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "virtual-kubelet-saladcloud.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end -}}

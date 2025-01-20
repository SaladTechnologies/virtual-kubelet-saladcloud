{{/*
Return the chart name, formatted for use as a label.
*/}}
{{- define "virtual-kubelet-saladcloud.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Return the chart name and version, formatted for use as a label.
*/}}
{{- define "virtual-kubelet-saladcloud.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Return the common labels used by components.
*/}}
{{- define "virtual-kubelet-saladcloud.labels" -}}
app.kubernetes.io/component: kubelet
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: {{ include "virtual-kubelet-saladcloud.name" . }}
app.kubernetes.io/version: {{ .Chart.AppVersion | default .Chart.Version }}
helm.sh/chart: {{ include "virtual-kubelet-saladcloud.chart" . }}
{{ include "virtual-kubelet-saladcloud.matchLabels" . }}
{{- with .Values.additionalLabels -}}
{{ toYaml . }}
{{- end -}}
{{- end -}}

{{/*
Return the common labels used by selectors.
*/}}
{{- define "virtual-kubelet-saladcloud.matchLabels" -}}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/name: {{ include "virtual-kubelet-saladcloud.name" . }}
{{- end -}}

{{/*
Return a fully qualified name.
*/}}
{{- define "virtual-kubelet-saladcloud.fullName" -}}
{{- if .Values.fullNameOverride -}}
{{- .Values.fullNameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- printf "%s-%s" .Release.Name .Values.name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s-%s" .Release.Name $name .Values.name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{/*
Return a fully qualified `ClusterRole` and `ClusterRoleBinding` name.
*/}}
{{- define "virtual-kubelet-saladcloud.clusterRoleName" -}}
{{- if .Values.clusterRoleNameOverride -}}
{{- .Values.clusterRoleNameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- include "virtual-kubelet-saladcloud.fullName" . -}}
{{- end -}}
{{- end -}}

{{/*
Return a fully qualified `ServiceAccount` name.
*/}}
{{- define "virtual-kubelet-saladcloud.serviceAccountName" -}}
{{- if .Values.serviceAccount.create -}}
{{- default (include "virtual-kubelet-saladcloud.fullName" .) .Values.serviceAccountNameOverride -}}
{{- else -}}
{{- default "default" .Values.serviceAccountNameOverride -}}
{{- end -}}
{{- end -}}

{{/*
Return the most appropriate `apiVersion` for RBAC.
*/}}
{{- define "virtual-kubelet-saladcloud.rbacApiVersion" -}}
{{- if .Capabilities.APIVersions.Has "rbac.authorization.k8s.io/v1" -}}
{{- print "rbac.authorization.k8s.io/v1" -}}
{{- else -}}
{{- print "rbac.authorization.k8s.io/v1beta1" -}}
{{- end -}}
{{- end -}}

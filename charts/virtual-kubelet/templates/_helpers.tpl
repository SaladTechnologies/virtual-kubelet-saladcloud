{{/*
Standard labels for helm resources
*/}}
{{- define "virtual-kubelet-saladcloud.labels" -}}
heritage: "{{ .Release.Service }}"
release: "{{ .Release.Name }}"
revision: "{{ .Release.Revision }}"
chart: "{{ .Chart.Name }}"
chartVersion: "{{ .Chart.Version }}"
{{- end -}}

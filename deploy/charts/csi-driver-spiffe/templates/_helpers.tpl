{{/* vim: set filetype=mustache: */}}
{{/*
Expand the name of the chart.
*/}}
{{- define "cert-manager-csi-driver-spiffe.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Create chart name and version as used by the chart label.
*/}}
{{- define "cert-manager-csi-driver-spiffe.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Common labels
*/}}
{{- define "cert-manager-csi-driver-spiffe.labels" -}}
app.kubernetes.io/name: {{ include "cert-manager-csi-driver-spiffe.name" . }}
helm.sh/chart: {{ include "cert-manager-csi-driver-spiffe.chart" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- if .Values.commonLabels }}
{{ toYaml .Values.commonLabels }}
{{- end }}
{{- end -}}

{{/*
Util function for generating the image URL based on the provided options.
IMPORTANT: This function is standarized across all charts in the cert-manager GH organization.
Any changes to this function should also be made in cert-manager, trust-manager, approver-policy, ...
See https://github.com/cert-manager/cert-manager/issues/6329 for a list of linked PRs.
*/}}
{{- define "image" -}}
{{- $defaultTag := index . 1 -}}
{{- with index . 0 -}}
{{- if .registry -}}{{ printf "%s/%s" .registry .repository }}{{- else -}}{{- .repository -}}{{- end -}}
{{- if .digest -}}{{ printf "@%s" .digest }}{{- else -}}{{ printf ":%s" (default $defaultTag .tag) }}{{- end -}}
{{- end }}
{{- end }}

{{/*
Backwards compatibility helper for driver image configuration.
Prefers legacy image format if set, otherwise uses new driverImage format.
*/}}
{{- define "driver-image-config" -}}
{{- $image := .Values.image | default dict -}}
{{- $repository := $image.repository | default dict -}}
{{- $digest := $image.digest | default dict -}}
{{- $config := dict -}}
{{- $_ := set $config "registry" ($image.registry | default .Values.driverImage.registry) -}}
{{- $_ := set $config "repository" ($repository.driver | default .Values.driverImage.repository) -}}
{{- $_ := set $config "tag" ($image.tag | default .Values.driverImage.tag) -}}
{{- $_ := set $config "digest" ($digest.driver | default .Values.driverImage.digest) -}}
{{- $_ := set $config "pullPolicy" ($image.pullPolicy | default .Values.driverImage.pullPolicy) -}}
{{- $config | toJson -}}
{{- end }}

{{/*
Backwards compatibility helper for approver image configuration.
Prefers legacy image format if set, otherwise uses new approverImage format.
*/}}
{{- define "approver-image-config" -}}
{{- $image := .Values.image | default dict -}}
{{- $repository := $image.repository | default dict -}}
{{- $digest := $image.digest | default dict -}}
{{- $config := dict -}}
{{- $_ := set $config "registry" ($image.registry | default .Values.approverImage.registry) -}}
{{- $_ := set $config "repository" ($repository.approver | default .Values.approverImage.repository) -}}
{{- $_ := set $config "tag" ($image.tag | default .Values.approverImage.tag) -}}
{{- $_ := set $config "digest" ($digest.approver | default .Values.approverImage.digest) -}}
{{- $_ := set $config "pullPolicy" ($image.pullPolicy | default .Values.approverImage.pullPolicy) -}}
{{- $config | toJson -}}
{{- end }}

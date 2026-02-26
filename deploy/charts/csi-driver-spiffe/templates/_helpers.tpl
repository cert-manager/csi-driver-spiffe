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
{{- $config := dict -}}
{{- if .Values.image -}}
  {{- /* Use legacy format if explicitly set */ -}}
  {{- $_ := set $config "registry" .Values.image.registry -}}
  {{- if .Values.image.repository -}}
    {{- if kindIs "map" .Values.image.repository -}}
      {{- /* Object format with driver/approver */ -}}
      {{- if .Values.image.repository.driver -}}
        {{- $_ := set $config "repository" .Values.image.repository.driver -}}
      {{- end -}}
    {{- else -}}
      {{- /* Simple string format */ -}}
      {{- $_ := set $config "repository" .Values.image.repository -}}
    {{- end -}}
  {{- end -}}
  {{- $_ := set $config "tag" .Values.image.tag -}}
  {{- if .Values.image.digest -}}
    {{- if kindIs "map" .Values.image.digest -}}
      {{- /* Object format with driver/approver */ -}}
      {{- if .Values.image.digest.driver -}}
        {{- $_ := set $config "digest" .Values.image.digest.driver -}}
      {{- end -}}
    {{- else -}}
      {{- /* Simple string format */ -}}
      {{- $_ := set $config "digest" .Values.image.digest -}}
    {{- end -}}
  {{- end -}}
  {{- $_ := set $config "pullPolicy" .Values.image.pullPolicy -}}
{{- else -}}
  {{- /* Use new format */ -}}
  {{- $config = .Values.driverImage -}}
{{- end -}}
{{- $config | toJson -}}
{{- end }}

{{/*
Backwards compatibility helper for approver image configuration.
Prefers legacy image format if set, otherwise uses new approverImage format.
*/}}
{{- define "approver-image-config" -}}
{{- $config := dict -}}
{{- if .Values.image -}}
  {{- /* Use legacy format if explicitly set */ -}}
  {{- $_ := set $config "registry" .Values.image.registry -}}
  {{- if .Values.image.repository -}}
    {{- if kindIs "map" .Values.image.repository -}}
      {{- /* Object format with driver/approver */ -}}
      {{- if .Values.image.repository.approver -}}
        {{- $_ := set $config "repository" .Values.image.repository.approver -}}
      {{- end -}}
    {{- else -}}
      {{- /* Simple string format */ -}}
      {{- $_ := set $config "repository" .Values.image.repository -}}
    {{- end -}}
  {{- end -}}
  {{- $_ := set $config "tag" .Values.image.tag -}}
  {{- if .Values.image.digest -}}
    {{- if kindIs "map" .Values.image.digest -}}
      {{- /* Object format with driver/approver */ -}}
      {{- if .Values.image.digest.approver -}}
        {{- $_ := set $config "digest" .Values.image.digest.approver -}}
      {{- end -}}
    {{- else -}}
      {{- /* Simple string format */ -}}
      {{- $_ := set $config "digest" .Values.image.digest -}}
    {{- end -}}
  {{- end -}}
  {{- $_ := set $config "pullPolicy" .Values.image.pullPolicy -}}
{{- else -}}
  {{- /* Use new format */ -}}
  {{- $config = .Values.approverImage -}}
{{- end -}}
{{- $config | toJson -}}
{{- end }}

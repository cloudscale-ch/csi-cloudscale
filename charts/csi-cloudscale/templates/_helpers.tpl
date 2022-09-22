{{/* Get Driver Name */}}
{{- define "csi-cloudscale.driver-name" -}}
{{- if .Values.nameOverride -}}
    {{ .Values.nameOverride }}
{{- else -}}
    {{ .Release.Name }}-csi-cloudscale
{{- end -}}
{{- end -}}

{{/* Get API Token Name */}}
{{- define "csi-cloudscale.api-token-name" -}}
{{ required "cloudscale.token.existingSecret" .Values.cloudscale.token.existingSecret }}
{{- end -}}

{{/* Get Controller Service Account Name*/}}
{{- define "csi-cloudscale.controller-service-account-name" -}}
{{- if .Values.controller.serviceAccountName -}}
    {{ .Values.controller.serviceAccountName }}
{{- else -}}
    {{ include "csi-cloudscale.driver-name" . }}-controller-sa
{{- end -}}
{{- end -}}

{{/* Get Node Service Account Name*/}}
{{- define "csi-cloudscale.node-service-account-name" -}}
{{- if .Values.node.serviceAccountName -}}
    {{ .Values.node.serviceAccountName }}
{{- else -}}
    {{ include "csi-cloudscale.driver-name" . }}-node-sa
{{- end -}}
{{- end -}}

{{/* When renderNamespace is true, include a Namespace definition. This is for emitting old-school YAMLs  */}}
{{- define "csi-cloudscale.namespace-in-yaml-manifest" -}}
{{/* See: https://github.com/helm/helm/issues/5465#issuecomment-473942223 */}}
{{- if .Values.renderNamespace -}}
namespace: {{ .Release.Namespace }}
{{- end -}}
{{- end -}}
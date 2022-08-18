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
{{- if .Values.cloudscale.token.existingSecret -}}
    {{ .Values.cloudscale.token.existingSecret -}}
{{- else -}}
    {{ .Release.Name }}-cloudscale-token
{{- end -}}
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

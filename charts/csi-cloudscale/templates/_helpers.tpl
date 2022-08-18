{{/* Get Driver Name */}}
{{- define "csi-cloudscale.driver-name" -}}
{{- if .Values.legacyName -}}
    csi-cloudscale
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

{{/* Get API Token Name */}}
{{- define "csi-cloudscale.api-token-name" -}}
{{- if .Values.cloudscale.token.existingSecret -}}
    {{ .Values.cloudscale.token.existingSecret }}
{{- else -}}
    {{ .Release.Name }}-cloudscale-token
{{- end -}}
{{- end -}}

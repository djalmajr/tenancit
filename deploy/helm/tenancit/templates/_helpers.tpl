{{- define "tenancit.name" -}}tenancit{{- end }}
{{- define "tenancit.fullname" -}}{{ .Release.Name }}{{- end }}
{{- define "tenancit.labels" -}}
app.kubernetes.io/name: {{ include "tenancit.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}
{{- define "tenancit.image" -}}
{{- $_ := required "image.digest is required and must be immutable" .Values.image.digest -}}
{{- printf "%s@%s" .Values.image.repository .Values.image.digest -}}
{{- end }}

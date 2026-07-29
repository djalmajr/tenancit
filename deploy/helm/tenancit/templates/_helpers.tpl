{{- define "tenancit.name" -}}tenancit{{- end }}
{{- define "tenancit.fullname" -}}{{ .Release.Name }}{{- end }}
{{- define "tenancit.labels" -}}
app.kubernetes.io/name: {{ include "tenancit.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}
{{- define "tenancit.image" -}}
{{- $_ := required "image.digest is required and must be immutable" .Values.image.digest -}}
{{- if not (regexMatch "^sha256:[0-9a-f]{64}$" .Values.image.digest) -}}
{{- fail "image.digest must be sha256 followed by 64 lowercase hexadecimal characters" -}}
{{- end -}}
{{- printf "%s@%s" .Values.image.repository .Values.image.digest -}}
{{- end }}
{{- define "tenancit.immutableImage" -}}
{{- if not (regexMatch "^[^@[:space:]]+@sha256:[0-9a-f]{64}$" .) -}}
{{- fail "personal images must use repository@sha256 followed by 64 lowercase hexadecimal characters" -}}
{{- end -}}
{{- . -}}
{{- end }}
{{- define "tenancit.validate" -}}
{{- if not (has .Values.adminAuth.mode (list "oidc" "legacy_shared_token")) -}}
{{- fail "adminAuth.mode must be oidc or legacy_shared_token" -}}
{{- end -}}
{{- if and .Values.ingress.enabled (eq .Values.adminAuth.mode "legacy_shared_token") -}}
{{- fail "legacy_shared_token admin auth must not be exposed through ingress" -}}
{{- end -}}
{{- $basePath := required "app.basePath is required" .Values.app.basePath -}}
{{- $ingressPath := required "ingress.path is required" .Values.ingress.path -}}
{{- if not (regexMatch "^/$|^/[A-Za-z0-9._~-]+(/[A-Za-z0-9._~-]+)*$" $basePath) -}}
{{- fail "app.basePath must be / or an absolute path with safe segments and no trailing slash" -}}
{{- end -}}
{{- if ne $ingressPath $basePath -}}
{{- fail "ingress.path must equal app.basePath" -}}
{{- end -}}
{{- end }}

{{/* Помощники имен и лейблов чарта rosterra. */}}

{{- define "rosterra.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "rosterra.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if ne $name .Release.Name -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "rosterra.labels" -}}
app.kubernetes.io/name: {{ include "rosterra.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end -}}

{{- define "rosterra.selectorLabels" -}}
app.kubernetes.io/name: {{ include "rosterra.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end -}}

{{/* Полное имя образа с registry/tag. */}}
{{- define "rosterra.image" -}}
{{- $registry := .ctx.Values.global.imageRegistry -}}
{{- $tag := .ctx.Values.global.imageTag | default .ctx.Chart.AppVersion -}}
{{- if $registry -}}{{ $registry }}/{{ .image }}:{{ $tag -}}
{{- else -}}{{ .image }}:{{ $tag -}}{{- end -}}
{{- end -}}

{{/* Имя компонента (для меток сервисов внутри чарта). */}}
{{- define "rosterra.component" -}}
{{ include "rosterra.fullname" .ctx }}-{{ .component }}
{{- end -}}

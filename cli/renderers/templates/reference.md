{{- if .Rich -}}
{{- range $i, $a := .Authors }}{{ if $i }}, {{ end }}{{ $a }}{{ end -}}, "{{ if .Title }}{{ .Title }}{{ end }}", {{ if .Journal }}*{{ .Journal }}*{{ end -}}
{{- if .Volume }}, vol. {{ .Volume }}{{ end -}}
{{- if .Issue }}, no. {{ .Issue }}{{ end -}}
{{- if .Page }}, pp. {{ .Page }}{{ end -}}
{{- if .Year }}, {{ .Year }}{{ end -}}
{{- if .DOI }}. doi: {{ .DOI }}{{ end -}}
.
{{- else -}}
{{ .Label }}
{{- end -}}

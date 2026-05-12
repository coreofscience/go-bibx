{{ frontMatter . }}

# {{ if .Title }}{{ .Title }}{{ else }}{{ .Label }}{{ end }}

{{ if .Abstract -}}
## Abstract

{{ wrap 80 .Abstract }}
{{ end -}}

{{ if .References -}}
{{ "" }}
## References

{{ range .References -}}
{{ if .Rich -}}
{{ render "reference.md" . | wrap 80 }}
{{ "" }}
{{ end -}}
{{ end -}}
{{ end -}}

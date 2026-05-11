{{ frontMatter . }}

# {{ if .Title }}{{ .Title }}{{ else }}{{ .Label }}{{ end }}

{{ if .Abstract -}}
## Abstract

{{ wrap 80 0 .Abstract }}
{{ end -}}

{{ if .References -}}
{{ "" }}
## References

{{ range .References -}}
- {{ if .Title }}{{ wrap 80 2 .Title }}{{ else }}{{ .Label }}{{ end }}
{{ end -}}
{{ end -}}

{{ frontmatter . }}

# {{ if .Title }}{{ .Title }}{{ else }}{{ .Label }}{{ end }}

{{ if .Abstract -}}
## Abstract

{{ wrap 80 .Abstract }}
{{ end -}}

{{ if .References -}}
## References

{{ range .References -}}
- {{ if .Title }}{{ .Title }}{{ else }}{{ .Label }}{{ end }}
{{ end -}}
{{ end -}}

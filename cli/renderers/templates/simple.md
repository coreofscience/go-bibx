# {{ if .Title }}{{ .Title }}{{ else }}{{ .Label }}{{ end }}
{{ if .GetPermalink -}}
**link**: [{{ .GetPermalink }}]({{ .GetPermalink }})
{{ end }}
{{ render "keywords.md" . | wrap 80 }}

{{ if .Abstract -}}
## Abstract

{{ wrap 80 .Abstract }}
{{ end -}}

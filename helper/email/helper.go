package email

import (
	"bytes"
	"embed"
	"html/template"
)

//go:embed *.html
var templateFS embed.FS

func ParseHtml(data any) (string, error) {

	tpl, err := template.ParseFS(templateFS, "email.html")
	if err != nil {
		return "", err
	}
	var body bytes.Buffer
	if err := tpl.Execute(&body, data); err != nil {
		return "", err
	}
	return body.String(), nil
}

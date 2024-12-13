package email

import (
	"bytes"
	"embed"
	_ "embed"
	"html/template"
)

//go:embed *.html
var templateFS embed.FS

type (
	IEmail interface {
		Send()
	}
	IParseHtml interface {
	}
)

func parseHtml(data any) (string, error) {

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

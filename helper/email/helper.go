package email

import (
	"bytes"
	"embed"
	"html/template"
	"sync"
)

//go:embed *.html
var templateFS embed.FS

var (
	emailTemplate     *template.Template
	emailTemplateErr  error
	emailTemplateOnce sync.Once
)

func ParseHTML(data any) (string, error) {
	emailTemplateOnce.Do(func() {
		emailTemplate, emailTemplateErr = template.ParseFS(templateFS, "email.html")
	})
	if emailTemplateErr != nil {
		return "", emailTemplateErr
	}
	var body bytes.Buffer
	if err := emailTemplate.Execute(&body, data); err != nil {
		return "", err
	}
	return body.String(), nil
}

func ParseHtml(data any) (string, error) {
	return ParseHTML(data)
}

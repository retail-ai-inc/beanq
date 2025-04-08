package email

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"html/template"

	"github.com/spf13/cast"
)

//go:embed *.html
var templateFS embed.FS

type (
	IEmail interface {
		From(from string)
		To(to string)
		Subject(subject string)
		Body(title, name, link string) error
		Send() error
	}
)

func NewEmail(ctx context.Context, keys ...string) (IEmail, error) {

	var email IEmail
	length := len(keys)

	if length <= 0 {
		return nil, errors.New("Email Init Error：parameter error")
	}
	//parameter will be : sendgrid `apiKey`
	if length > 0 || length < 4 {
		email = NewSendGrid(ctx, keys[0])
	}
	//parameter will be: host,port,user,password
	if length >= 4 {
		email = NewGoEmail(keys[0], cast.ToInt(keys[1]), keys[2], keys[3])
	}

	return email, nil
}

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

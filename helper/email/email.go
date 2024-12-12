package email

import (
	"bytes"
	"context"
	_ "embed"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"github.com/spf13/viper"
	"html/template"
)

//go:embed email.html
var emailTpl string

// If possible, we can expand more fields
type EmbedData struct {
	Title string
	Name  string
	Link  string
}

func DefaultSend(ctx context.Context, toName, toAddress string, data *EmbedData) (statusCode int, body string, headers map[string][]string, err error) {

	from := mail.NewEmail(viper.GetString("email.fromName"), viper.GetString("email.fromAddress"))
	to := mail.NewEmail(toName, toAddress)
	key := viper.GetString("email.key")

	return Send(ctx, from, to, key, data)
}

func Send(ctx context.Context, from, to *mail.Email, apiKey string, data *EmbedData) (statusCode int, body string, headers map[string][]string, err error) {

	subject := "Active Email"
	plainTextContent := ""
	htmlContent, err := parseHtml(data)
	if err != nil {
		return 0, "", nil, err
	}

	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)

	client := sendgrid.NewSendClient(apiKey)
	response, err := client.SendWithContext(ctx, message)

	if err != nil {
		return
	}
	return response.StatusCode, response.Body, response.Headers, nil

}

func parseHtml(data *EmbedData) (string, error) {

	tpl, err := template.New("email-template").Parse(emailTpl)
	if err != nil {
		return "", err
	}

	var body bytes.Buffer
	if err := tpl.Execute(&body, data); err != nil {
		return "", err
	}
	return body.String(), nil
}

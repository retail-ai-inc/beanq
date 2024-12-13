package email

import (
	"context"
	"errors"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

type SendGrid struct {
	from    *mail.Email
	to      *mail.Email
	subject string
	body    string
	apiKey  string
	ctx     context.Context
}

func NewSendGrid(ctx context.Context, apiKey string) *SendGrid {
	return &SendGrid{
		apiKey: apiKey,
		ctx:    ctx,
	}
}

func (t *SendGrid) From(from string) {
	t.from = mail.NewEmail("Retail-AI", from)
}

func (t *SendGrid) To(to string) {
	t.to = mail.NewEmail("Retail-AI", to)
}

func (t *SendGrid) Subject(subject string) {
	t.subject = subject
}

func (t *SendGrid) Body(title, name, link string) error {

	body, err := parseHtml(map[string]any{"Title": title, "Name": name, "Link": link})
	if err != nil {
		return err
	}
	t.body = body
	return nil
}

func (t *SendGrid) Send() error {

	message := mail.NewSingleEmail(t.from, t.subject, t.to, "", t.body)

	client := sendgrid.NewSendClient(t.apiKey)
	response, err := client.SendWithContext(t.ctx, message)

	if err != nil {
		return err
	}
	if response.StatusCode != 200 {
		return errors.New(response.Body)
	}
	return nil

}

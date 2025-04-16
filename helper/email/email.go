package email

import (
	"context"
	"errors"
	"time"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"gopkg.in/gomail.v2"
)

type Option struct {
	Host     string
	Port     int
	User     string
	Password string
	ApiKey   string
}

type Options func(option *Option)

func WithHost(host string) Options {
	return func(option *Option) {
		option.Host = host
	}
}
func WithPort(port int) Options {
	return func(option *Option) {
		option.Port = port
	}
}
func WithUser(user string) Options {
	return func(option *Option) {
		option.User = user
	}
}
func WithPassword(password string) Options {
	return func(option *Option) {
		option.Password = password
	}
}
func WithApiKey(apiKey string) Options {
	return func(option *Option) {
		option.ApiKey = apiKey
	}
}

type Email struct {
	client  any
	ctx     context.Context
	from    string
	to      string
	subject string
	body    string
	date    time.Time
}

func (t *Email) From(from string) {
	t.from = from
}
func (t *Email) To(to string) {
	t.to = to
}
func (t *Email) Subject(subject string) {
	t.subject = subject
}
func (t *Email) Date(date time.Time) {
	t.date = date
}
func (t *Email) InviteHtmlBody(title, name, link string) error {
	body, err := ParseHtml(map[string]any{"Title": title, "Name": name, "Link": link})
	if err != nil {
		return err
	}
	t.body = body
	return nil
}
func (t *Email) TextBody(body string) {
	t.body = body
}
func (t *Email) Send() error {
	if t.client == nil {
		return errors.New("client is nil")
	}

	if v, ok := t.client.(*gomail.Dialer); ok {
		msg := gomail.NewMessage()
		msg.SetHeader("From", t.from)
		msg.SetDateHeader("Date", t.date)
		msg.SetHeader("To", t.to)
		msg.SetHeader("Subject", t.subject)
		msg.SetBody("text/html", t.body)
		return v.DialAndSend(msg)
	}
	if v, ok := t.client.(*sendgrid.Client); ok {

		from := mail.NewEmail("Retail-AI", t.from)
		to := mail.NewEmail("Retail-AI", t.to)
		msg := mail.NewSingleEmail(from, t.subject, to, "", t.body)
		_, err := v.Send(msg)
		if err != nil {
			return err
		}
	}
	return nil
}

func NewClient(options ...Options) *Email {

	if len(options) == 0 {
		return nil
	}
	var client any
	opt := &Option{}
	for _, option := range options {
		option(opt)
	}
	if opt.Host != "" && opt.Port != 0 && opt.User != "" && opt.Password != "" {
		client = gomail.NewDialer(opt.Host, opt.Port, opt.User, opt.Password)
	} else if opt.ApiKey != "" {
		client = sendgrid.NewSendClient(opt.ApiKey)
	}
	email := Email{
		client: client,
		ctx:    nil,
		date:   time.Now(),
	}
	return &email
}

func NewGoEmail(host string, port int, username string, password string) *Email {

	return NewClient(WithHost(host), WithPort(port), WithUser(username), WithPassword(password))
}

func NewSendGrid(apiKey string) *Email {

	return NewClient(WithApiKey(apiKey))

}

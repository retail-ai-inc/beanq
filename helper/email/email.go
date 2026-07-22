package email

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/sendgrid/sendgrid-go"
	sendgridmail "github.com/sendgrid/sendgrid-go/helpers/mail"
	"gopkg.in/gomail.v2"
)

type Option struct {
	Host     string
	Port     int
	User     string
	Password string
	APIKey   string
}

type Options func(option *Option)

func WithHost(host string) Options {
	return func(option *Option) { option.Host = host }
}

func WithPort(port int) Options {
	return func(option *Option) { option.Port = port }
}

func WithUser(user string) Options {
	return func(option *Option) { option.User = user }
}

func WithPassword(password string) Options {
	return func(option *Option) { option.Password = password }
}

func WithAPIKey(apiKey string) Options {
	return func(option *Option) { option.APIKey = apiKey }
}

func WithApiKey(apiKey string) Options {
	return WithAPIKey(apiKey)
}

type Email struct {
	provider provider
	fromName string
	from     string
	to       string
	subject  string
	body     string
	html     bool
	date     time.Time
}

func (t *Email) From(from string) {
	t.from = strings.TrimSpace(from)
}

func (t *Email) FromName(name string) {
	t.fromName = strings.TrimSpace(name)
}

func (t *Email) To(to string) {
	t.to = strings.TrimSpace(to)
}

func (t *Email) Subject(subject string) {
	t.subject = strings.TrimSpace(subject)
}

func (t *Email) Date(date time.Time) {
	t.date = date
}

func (t *Email) InviteHtmlBody(title, name, link string) error {
	body, err := ParseHTML(map[string]any{"Title": title, "Name": name, "Link": link})
	if err != nil {
		return err
	}
	t.body = body
	t.html = true
	return nil
}

func (t *Email) TextBody(body string) {
	t.body = body
	t.html = false
}

func (t *Email) Send() error {
	return t.SendContext(context.Background())
}

func (t *Email) SendContext(ctx context.Context) error {
	if t == nil || t.provider == nil {
		return errors.New("email provider is not configured")
	}
	if err := t.validate(); err != nil {
		return err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	message := Message{
		FromName: t.fromName,
		From:     t.from,
		To:       t.to,
		Subject:  t.subject,
		Date:     t.date,
	}
	if t.html {
		message.HTML = t.body
	} else {
		message.Text = t.body
	}
	if message.FromName == "" {
		message.FromName = "Retail-AI"
	}
	if message.Date.IsZero() {
		message.Date = time.Now()
	}
	if err := t.provider.Send(ctx, message); err != nil {
		return fmt.Errorf("send email: %w", err)
	}
	return nil
}

func (t *Email) validate() error {
	if _, err := mail.ParseAddress(t.from); err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}
	if _, err := mail.ParseAddress(t.to); err != nil {
		return fmt.Errorf("invalid recipient address: %w", err)
	}
	if t.subject == "" {
		return errors.New("email subject is required")
	}
	if t.body == "" {
		return errors.New("email body is required")
	}
	return nil
}

func NewClient(options ...Options) (*Email, error) {
	if len(options) == 0 {
		return nil, errors.New("email provider configuration is required")
	}
	config := Option{}
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}

	var configuredProvider provider
	switch {
	case strings.TrimSpace(config.Host) != "" || config.Port != 0 || strings.TrimSpace(config.User) != "":
		if strings.TrimSpace(config.Host) == "" || config.Port < 1 || config.Port > 65535 || strings.TrimSpace(config.User) == "" {
			return nil, errors.New("SMTP host, valid port, and user are required")
		}
		dialer := gomail.NewDialer(config.Host, config.Port, config.User, config.Password)
		configuredProvider = smtpProvider{dialer: dialer}
	case strings.TrimSpace(config.APIKey) != "":
		configuredProvider = sendGridProvider{client: sendgrid.NewSendClient(config.APIKey)}
	default:
		return nil, errors.New("SMTP or SendGrid configuration is required")
	}

	return &Email{provider: configuredProvider, date: time.Now()}, nil
}

func NewGoEmail(host string, port int, username, password string) (*Email, error) {
	return NewClient(WithHost(host), WithPort(port), WithUser(username), WithPassword(password))
}

func NewSendGrid(apiKey string) (*Email, error) {
	return NewClient(WithAPIKey(apiKey))
}

type smtpProvider struct {
	dialer *gomail.Dialer
}

func (p smtpProvider) Send(ctx context.Context, message Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	mailMessage := gomail.NewMessage()
	mailMessage.SetAddressHeader("From", message.From, message.FromName)
	mailMessage.SetDateHeader("Date", message.Date)
	mailMessage.SetHeader("To", message.To)
	mailMessage.SetHeader("Subject", message.Subject)
	if message.HTML != "" {
		mailMessage.SetBody("text/html", message.HTML)
	} else {
		mailMessage.SetBody("text/plain", message.Text)
	}
	return p.dialer.DialAndSend(mailMessage)
}

type sendGridProvider struct {
	client *sendgrid.Client
}

func (p sendGridProvider) Send(ctx context.Context, message Message) error {
	from := sendgridmail.NewEmail(message.FromName, message.From)
	to := sendgridmail.NewEmail("", message.To)
	mailMessage := sendgridmail.NewSingleEmail(from, message.Subject, to, message.Text, message.HTML)
	response, err := p.client.SendWithContext(ctx, mailMessage)
	if err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("SendGrid returned status %d: %.512s", response.StatusCode, response.Body)
	}
	return nil
}

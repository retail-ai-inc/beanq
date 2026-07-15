package email

import (
	"context"
	"time"
)

type Message struct {
	FromName string
	From     string
	To       string
	Subject  string
	Text     string
	HTML     string
	Date     time.Time
}

type provider interface {
	Send(ctx context.Context, message Message) error
}

type Sender interface {
	From(from string)
	FromName(name string)
	To(to string)
	Subject(subject string)
	Date(date time.Time)
	TextBody(body string)
	InviteHtmlBody(title, name, link string) error
	Send() error
	SendContext(ctx context.Context) error
}

var _ Sender = (*Email)(nil)

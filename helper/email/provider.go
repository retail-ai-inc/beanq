package email

import "time"

// Sender is the test-friendly boundary for e-mail integrations.
type Sender interface {
	From(from string)
	To(to string)
	Subject(subject string)
	Date(date time.Time)
	TextBody(body string)
	InviteHtmlBody(title, name, link string) error
	Send() error
}

var _ Sender = (*Email)(nil)

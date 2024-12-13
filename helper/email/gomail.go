package email

import (
	"crypto/tls"
	"gopkg.in/gomail.v2"
	"time"
)

type GoEmail struct {
	msg    *gomail.Message
	dialer *gomail.Dialer
}

func NewEmail(host string, port int, username string, password string) *GoEmail {

	dialer := gomail.NewDialer(host, port, username, password)
	dialer.TLSConfig = &tls.Config{
		InsecureSkipVerify: true,
	}
	return &GoEmail{
		msg:    gomail.NewMessage(),
		dialer: dialer,
	}
}

func (t *GoEmail) From(from string) {
	t.msg.SetHeader("From", from)
}

func (t *GoEmail) To(to string) {
	t.msg.SetHeader("To", to)
}

func (t *GoEmail) Subject(subject string) {
	t.msg.SetHeader("Subject", subject)
}

func (t *GoEmail) Date() {
	t.msg.SetDateHeader("Date", time.Now())
}

func (t *GoEmail) Body(title, name, link string) error {

	body, err := parseHtml(map[string]any{"Title": title, "Name": name, "Link": link})
	if err != nil {
		return err
	}

	t.msg.SetBody("text/html", body)
	return nil
}

func (t *GoEmail) Send() error {

	if err := t.dialer.DialAndSend(t.msg); err != nil {
		return err
	}
	return nil
}

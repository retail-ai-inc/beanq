package email

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type fakeProvider struct {
	message Message
	err     error
}

func (p *fakeProvider) Send(_ context.Context, message Message) error {
	p.message = message
	return p.err
}

func TestSendValidatesAndPassesAlertMessage(t *testing.T) {
	provider := &fakeProvider{}
	client := &Email{provider: provider}
	client.From("alerts@example.com")
	client.FromName("BeanQ")
	client.To("operator@example.com")
	client.Subject("BeanQ alert")
	client.TextBody("queue processing failed")

	if err := client.SendContext(context.Background()); err != nil {
		t.Fatalf("SendContext returned error: %v", err)
	}
	if provider.message.FromName != "BeanQ" || provider.message.Subject != "BeanQ alert" || provider.message.Text != "queue processing failed" || provider.message.HTML != "" {
		t.Fatalf("unexpected message: %#v", provider.message)
	}
	if provider.message.Date.IsZero() {
		t.Fatal("message date was not set")
	}
}

func TestSendRejectsInvalidMessage(t *testing.T) {
	client := &Email{provider: &fakeProvider{}}
	client.From("invalid")
	client.To("operator@example.com")
	client.Subject("alert")
	client.TextBody("body")
	if err := client.Send(); err == nil || !strings.Contains(err.Error(), "sender") {
		t.Fatalf("expected sender validation error, got %v", err)
	}
}

func TestSendWrapsProviderError(t *testing.T) {
	providerErr := errors.New("provider unavailable")
	client := &Email{provider: &fakeProvider{err: providerErr}, from: "alerts@example.com", to: "operator@example.com", subject: "alert", body: "body"}
	if err := client.Send(); !errors.Is(err, providerErr) {
		t.Fatalf("expected wrapped provider error, got %v", err)
	}
}

func TestNewClientValidatesProviderConfiguration(t *testing.T) {
	if _, err := NewClient(); err == nil {
		t.Fatal("expected missing provider error")
	}
	if _, err := NewGoEmail("smtp.example.com", 70000, "user", "password"); err == nil {
		t.Fatal("expected invalid SMTP port error")
	}
	if _, err := NewSendGrid(""); err == nil {
		t.Fatal("expected missing SendGrid key error")
	}
}

func TestInviteTemplateEscapesValues(t *testing.T) {
	client := &Email{provider: &fakeProvider{}}
	if err := client.InviteHtmlBody("Invite", "<script>alert(1)</script>", "https://example.com/?a=1&b=2"); err != nil {
		t.Fatalf("InviteHtmlBody returned error: %v", err)
	}
	if strings.Contains(client.body, "<script>") || !strings.Contains(client.body, "&lt;script&gt;") {
		t.Fatalf("template did not escape recipient name: %s", client.body)
	}
}

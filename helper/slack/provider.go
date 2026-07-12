package xslack

import "context"

// Sender is the test-friendly boundary for Slack integrations.
type Sender interface {
	Channel(channel string)
	Color(color string)
	Footer(footer string)
	FooterIcon(icon string)
	Title(title string)
	TitleLink(titleLink string)
	Send(ctx context.Context, field ...Field) error
}

var _ Sender = (*Client)(nil)

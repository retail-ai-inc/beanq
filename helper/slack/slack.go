//go:generate fzgen -o ../../test/fuzz/slackfuzz_test.go
package xslack

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/slack-go/slack"
	"github.com/spf13/cast"
)

const (
	// Attachment color
	Good      string = string(slack.StyleDefault)
	Primary   string = string(slack.StylePrimary)
	Danger    string = string(slack.StyleDanger)
	Warning   string = "#ffc107"
	Info      string = "#0dcaf0"
	Secondary string = "#6c757d"
	Success   string = "#198754"
	Dark      string = "#212529"
)

type Client struct {
	client     *slack.Client
	channel    string
	color      string
	title      string
	titleLink  string
	markDownIn []string
	footer     string
	footerIcon string
	preText    string
	text       string
}

var (
	client Client
	once   sync.Once
)

func NewClient(botAuthToken string) *Client {
	once.Do(func() {
		client.client = slack.New(botAuthToken, slack.OptionDebug(true))
		client.color = Warning
		client.markDownIn = []string{"title", "text", "pretext"}
		client.footer = "Send Notice By Beanq"
		client.title = "Beanq Notice"
	})
	return &client
}

// Channel The channel where the message will be sent
func (t *Client) Channel(channel string) {
	t.channel = channel
}

// Color The color of the sidebar prompt
func (t *Client) Color(color string) {
	t.color = color
}

// Footer The footer of the message
func (t *Client) Footer(footer string) {
	t.footer = footer
}

func (t *Client) FooterIcon(icon string) {
	t.footerIcon = icon
}

func (t *Client) Title(title string) {
	t.title = title
}

func (t *Client) TitleLink(titleLink string) {
	t.titleLink = titleLink
}

// TeamInfo  need  team:read
func (t *Client) TeamInfo() (*slack.TeamInfo, error) {
	return t.client.GetTeamInfo()
}

type Field = slack.AttachmentField

func (t *Client) Send(ctx context.Context, field ...Field) error {

	if t.channel == "" {
		return errors.New("Channel Err:[channel is required]")
	}

	fields := make([]slack.AttachmentField, len(field))
	fields = append(fields, field...)

	now := cast.ToString(time.Now().Unix())
	attachment := slack.Attachment{
		Color:      t.color,
		Title:      t.title,
		TitleLink:  t.titleLink,
		Pretext:    t.preText,
		Text:       t.text,
		Fields:     fields,
		MarkdownIn: t.markDownIn,
		Footer:     t.footer,
		FooterIcon: t.footerIcon,
		Ts:         json.Number(now),
	}
	_, _, err := t.client.PostMessageContext(ctx, t.channel, slack.MsgOptionAttachments(attachment))
	return err
}

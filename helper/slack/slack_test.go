package xslack

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/slack-go/slack"
	"github.com/spf13/cast"
)

func TestWebHook(t *testing.T) {

	hookUrl := ""
	if hookUrl == "" {
		t.Skip("Suitable for local testing")
	}
	tm := cast.ToString(time.Now().Unix())
	attachment := slack.Attachment{
		Color:     Danger,
		Title:     "Test",
		TitleLink: "www.google.com",
		Pretext:   "_*Super Bot Message*_",
		Text:      "aaaaaa",
		Fields: []slack.AttachmentField{
			{
				Title: "Title",
				Value: ":smile:Value",
				Short: true,
			},
			{
				Title: "Title2",
				Value: "请访问<https://www.example.com|网站>",
				Short: true,
			},
		},
		MarkdownIn: []string{"text", "pretext"},
		Footer:     "send notice by Beanq",
		FooterIcon: "",
		Ts:         json.Number(tm),
	}
	msg := slack.WebhookMessage{Attachments: []slack.Attachment{attachment}}
	err := slack.PostWebhook(hookUrl, &msg)

	if err != nil {
		t.Fatal(err)
	}

}

func TestSlack(t *testing.T) {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	channelId := ""
	token := "" // bot token
	if channelId == "" || token == "" {
		t.Skip("Suitable for local testing")
	}
	client := NewClient(token)

	client.Channel(channelId)
	client.Color(Danger)
	err := client.Send(ctx, Field{
		Title: "Test",
		Value: "aa",
		Short: true,
	}, Field{
		Title: "Test2",
		Value: "bb",
		Short: true,
	})

	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Success")
}

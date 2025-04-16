package email

import (
	"fmt"
	"testing"
)

func TestSendNormal(t *testing.T) {

	ge := NewGoEmail("smtp.126.com", 25, "bandaoqiu1@126.com", "")

	ge.From("bandaoqiu1@126.com")
	ge.Subject("Retail Admin Invitation")
	if err := ge.InviteHtmlBody("Send Email", "10223062kong_liangliang@cn.tre-inc.com", "https://google.com"); err != nil {
		fmt.Println(err)
	}

	ge.To("bandaoqiu1@126.com")
	if err := ge.Send(); err != nil {
		t.Error(err)
	}

}

func TestSendGrid(t *testing.T) {

	client := NewSendGrid("xxxxxxx")
	client.From("noreply@retail-ai.jp")
	client.To("10223062kong_liangliang@cn.tre-inc.com")
	client.Subject("Retail Admin Invitation")
	client.TextBody("aaaaa")

	if err := client.Send(); err != nil {
		t.Error(err)
	}
}

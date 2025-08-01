package email

import (
	"fmt"
	"testing"
)

func TestSendNormal(t *testing.T) {

	password := ""
	username := "bandaoqiu1@126.com"
	port := 25
	host := "smtp.126.com"
	if password == "" {
		t.Skip("Please enter the correct password to test")
	}
	ge := NewGoEmail(host, port, username, password)

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
	apikey := ""
	if apikey == "" {
		t.Skip("Please enter the correct apikey to test")
	}
	client := NewSendGrid(apikey)
	client.From("noreply@retail-ai.jp")
	client.To("10223062kong_liangliang@cn.tre-inc.com")
	client.Subject("Retail Admin Invitation")
	client.TextBody("aaaaa")

	if err := client.Send(); err != nil {
		t.Error(err)
	}
}

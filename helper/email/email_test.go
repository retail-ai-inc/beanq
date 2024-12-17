package email

import (
	"context"
	"fmt"
	"testing"
)

func TestSendNormal(t *testing.T) {

	ge := NewGoEmail("smtp.126.com", 25, "kllztt@126.com", "DSYcnDeJs2wnGHEW")

	ge.From("kllztt@126.com")
	ge.To("10223062kong_liangliang@cn.tre-inc.com")
	ge.Subject("Retail Admin Invitation")
	if err := ge.Body("Send Email", "10223062kong_liangliang@cn.tre-inc.com", "https://google.com"); err != nil {
		fmt.Println(err)
	}

	if err := ge.Send(); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("success")
	}

}

func TestSendGrid(t *testing.T) {

	client := NewSendGrid(context.Background(), "xxxxxxx")
	client.From("noreply@retail-ai.jp")
	client.To("10223062kong_liangliang@cn.tre-inc.com")
	client.Subject("Retail Admin Invitation")

	_ = client.Body("Send Email", "10223062kong_liangliang@cn.tre-inc.com", "https://google.com")

	if err := client.Send(); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("success")
	}
}

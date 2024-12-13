package email

import (
	"context"
	"fmt"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"log"
	"testing"
)

func TestSendNormal(t *testing.T) {

	ge := NewEmail("smtp.126.com", 25, "kllztt@126.com", "DSYcnDeJs2wnGHEW")

	ge.From("kllztt@126.com")
	ge.To("10223062kong_liangliang@cn.tre-inc.com")
	ge.Subject("Retail AI Admin Invitation")
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

	ctx := context.Background()
	from := mail.NewEmail("Retail-AI", "noreply@retail-ai.jp")
	to := mail.NewEmail("KongLiangLiang", "10223062kong_liangliang@cn.tre-inc.com")
	apiKey := "xxxxx"
	data := &EmbedData{
		Title: "Test Send Email",
		Name:  "TRIAL",
		Link:  "http://www.trial.jp",
	}
	code, body, header, err := Send(ctx, from, to, apiKey, data)
	if err != nil {
		log.Fatalf("send email err:%+v \n", err)
	}
	fmt.Printf("Code:%+v,Body:%+v,Header:%+v \n", code, body, header)
}

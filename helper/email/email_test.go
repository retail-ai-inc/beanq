package email

import (
	"context"
	"fmt"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"log"
	"testing"
)

func TestSend(t *testing.T) {

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

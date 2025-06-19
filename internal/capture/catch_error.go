package capture

import (
	"slices"
	"time"

	"github.com/retail-ai-inc/beanq/v3/helper/email"
	"github.com/retail-ai-inc/beanq/v3/helper/logger"
	xslack "github.com/retail-ai-inc/beanq/v3/helper/slack"
	"github.com/spf13/cast"
	"golang.org/x/net/context"
)

type (
	CatchType string

	Catch struct {
		catchType CatchType
		channel   string
		topic     []string
		rule      *AlertRule
		config    *Config
	}
	Channel struct {
		Channel string
		Topic   []string
	}

	AlertRule struct {
		When []CatchType
		If   []If
		Then []Then
	}
)

var (
	System CatchType = "system"
	Dlq    CatchType = "dlq"
	Fail   CatchType = "fail"
)

// When
// It will be optimized in the later stage
func (t CatchType) When(config *Config) *Catch {

	if config == nil {
		return nil
	}

	whens := make([]CatchType, 0)
	for _, w := range config.Rule.When {
		whens = append(whens, CatchType(w.Value))
	}

	capCfg := AlertRule{
		When: whens,
		If:   config.Rule.If,
		Then: config.Rule.Then,
	}

	if len(capCfg.When) <= 0 {
		return nil
	}
	// boundary condition
	if slices.Contains(capCfg.When, t) {
		return &Catch{
			catchType: t,
			rule:      &capCfg,
			config:    config,
		}
	}
	return nil
}

func (t *Catch) If(chl *Channel) *Catch {

	if t == nil {
		return t
	}
	if t.catchType == System {
		return t
	}

	// if the channel is empty, return directly without sending an email or slack
	if chl.Channel == "" {
		return nil
	}

	for _, v := range t.rule.If {

		if v.Key != chl.Channel {
			continue
		}
		if len(v.Topic) <= 0 {
			return &Catch{
				channel:   chl.Channel,
				topic:     []string{},
				catchType: t.catchType,
				config:    t.config,
			}
		}

		for _, vt := range chl.Topic {
			for _, topic := range v.Topic {
				if vt == topic.Topic {
					return &Catch{
						channel:   chl.Channel,
						topic:     []string{vt},
						catchType: t.catchType,
						config:    t.config,
					}
				}
			}
		}
	}

	return nil
}

func (t *Catch) Then(err error) {

	if t == nil {
		return
	}

	if err == nil {
		return
	}

	for _, then := range t.rule.Then {
		if then.Key == "email" {
			host := t.config.SMTP.Host
			port := t.config.SMTP.Port
			user := t.config.SMTP.User
			password := t.config.SMTP.Password
			if host == "" || port == "" || user == "" || password == "" {
				continue
			}

			client := email.NewGoEmail(host, cast.ToInt(port), user, password)
			client.From(user)
			client.Subject("Notify")
			client.TextBody(err.Error())
			client.To(then.Value)
			if err := client.Send(); err == nil {
				continue
			} else {
				logger.New().Error(err)
			}
			if t.config.SendGrid.Key == "" {
				continue
			}

			client = email.NewSendGrid(t.config.SendGrid.Key)
			client.From(t.config.SendGrid.FromAddress)
			client.Subject("Notify")
			client.TextBody(err.Error())
			client.To(then.Value)
			if err := client.Send(); err != nil {
				logger.New().Error(err)
				continue
			}
			logger.New().Error(err)
		}
		if then.Key == "slack" {
			if t.config.Slack.BotAuthToken == "" {
				continue
			}
			if then.Parameters.Channel == "" && then.Parameters.WorkSpace == "" {
				continue
			}
			xclient := xslack.NewClient(t.config.Slack.BotAuthToken)
			xclient.Channel(then.Parameters.Channel)
			xclient.Color(xslack.Danger)
			ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
			if err := xclient.Send(ctx, xslack.Field{Title: "Beanq Error", Value: err.Error(), Short: true}); err != nil {
				logger.New().Error(err)
			}
		}
	}
}

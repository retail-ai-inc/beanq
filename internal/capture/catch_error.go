package capture

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/retail-ai-inc/beanq/v4/helper/email"
	"github.com/retail-ai-inc/beanq/v4/helper/logger"
	xslack "github.com/retail-ai-inc/beanq/v4/helper/slack"
	"github.com/spf13/cast"
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

type alertDispatcher struct {
	queue chan func()
}

func newAlertDispatcher(workers, capacity int) *alertDispatcher {
	dispatcher := &alertDispatcher{queue: make(chan func(), capacity)}
	for range max(workers, 1) {
		go func() {
			for job := range dispatcher.queue {
				job()
			}
		}()
	}
	return dispatcher
}

func (d *alertDispatcher) submit(job func()) bool {
	select {
	case d.queue <- job:
		return true
	default:
		return false
	}
}

var defaultAlertDispatcher = newAlertDispatcher(4, 128)

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
	if t == nil || err == nil {
		return
	}
	message := err.Error()
	if !defaultAlertDispatcher.submit(func() { t.send(message) }) {
		logger.New().Error(fmt.Errorf("BeanQ alert queue is full; dropping alert"))
	}
}

func (t *Catch) send(message string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, then := range t.rule.Then {
		if then.Key == "email" {
			host := t.config.Email.SMTP.Host
			port := t.config.Email.SMTP.Port
			user := t.config.Email.SMTP.User
			password := t.config.Email.SMTP.Password
			if host != "" && port != "" && user != "" {
				client, clientErr := email.NewGoEmail(host, cast.ToInt(port), user, password)
				if clientErr != nil {
					logger.New().Error(clientErr)
				} else {
					client.From(user)
					client.Subject("BeanQ alert")
					client.TextBody(message)
					client.To(then.Value)
					if sendErr := client.SendContext(ctx); sendErr == nil {
						continue
					} else {
						logger.New().Error(sendErr)
					}
				}
			}
			if t.config.Email.SendGrid.Key == "" {
				continue
			}
			client, clientErr := email.NewSendGrid(t.config.Email.SendGrid.Key)
			if clientErr != nil {
				logger.New().Error(clientErr)
				continue
			}
			client.From(t.config.Email.SendGrid.FromAddress)
			client.FromName(t.config.Email.SendGrid.FromName)
			client.Subject("BeanQ alert")
			client.TextBody(message)
			client.To(then.Value)
			if sendErr := client.SendContext(ctx); sendErr != nil {
				logger.New().Error(sendErr)
			}
		}
		if then.Key == "slack" {
			if t.config.Slack.BotAuthToken == "" || (then.Parameters.Channel == "" && then.Parameters.WorkSpace == "") {
				continue
			}
			client := xslack.NewClient(t.config.Slack.BotAuthToken)
			client.Channel(then.Parameters.Channel)
			client.Color(xslack.Danger)
			if sendErr := client.Send(ctx, xslack.Field{Title: "Beanq Error", Value: message, Short: true}); sendErr != nil {
				logger.New().Error(sendErr)
			}
		}
	}
}

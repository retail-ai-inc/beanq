package capture

import (
	"slices"

	"github.com/retail-ai-inc/beanq/v3/helper/email"
	"github.com/retail-ai-inc/beanq/v3/helper/logger"
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

var (
	System CatchType = "system"
	Dlq    CatchType = "dlq"
	Fail   CatchType = "fail"
)

// When
// It will be optimized in the later stage
func (t CatchType) When(config *Config) *Catch {

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

	var nerr error
	host := t.config.SMTP.Host
	port := t.config.SMTP.Port
	user := t.config.SMTP.User
	password := t.config.SMTP.Password

	if host != "" && port != "" && user != "" && password != "" {
		client := email.NewGoEmail(host, cast.ToInt(port), user, password)
		client.From(user)
		client.Subject("Test Notify")
		client.TextBody(err.Error())
		for _, then := range t.rule.Then {
			client.To(then.Value)
			nerr = client.Send()
		}
	}
	if nerr != nil {
		if t.config.SendGrid.Key != "" {
			client := email.NewSendGrid(t.config.SendGrid.Key)
			client.From(user)
			client.Subject("Test Notify")
			client.TextBody(err.Error())
			for _, then := range t.rule.Then {
				client.To(then.Value)
				nerr = client.Send()
			}
		}
	}

	if nerr != nil {
		logger.New().Error(nerr)
	}

}

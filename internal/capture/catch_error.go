package capture

import (
	"slices"
	"time"

	"github.com/retail-ai-inc/beanq/v3/helper/email"
	"github.com/retail-ai-inc/beanq/v3/helper/logger"
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
		If   []Channel
		Then string
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
		nw, err := cast.ToStringMapStringE(w)
		if err == nil {
			if v, ok := nw["key"]; ok {
				whens = append(whens, CatchType(v))
			}
		}
	}
	ifs := make([]Channel, 0)
	ch := Channel{
		Channel: "",
		Topic:   nil,
	}
	for _, v := range config.Rule.If {
		ch = Channel{
			Channel: "",
			Topic:   nil,
		}
		if nv, ok := v.(map[string]any); ok {
			if nv, ok := nv["key"]; ok {
				ch.Channel = nv.(string)
			}
			if nv, ok := nv["topic"]; ok {
				if nv, ok := nv.([]any); ok {
					for _, vt := range nv {
						if vt, ok := vt.(map[string]any); ok {
							if vt, ok := vt["topic"]; ok {
								ch.Topic = append(ch.Topic, vt.(string))
							}
						}
					}
				}
			}
		}
		ifs = append(ifs, ch)
	}

	capCfg := AlertRule{
		When: whens,
		If:   ifs,
		Then: "",
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

		if v.Channel != chl.Channel {
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
			if slices.Contains(v.Topic, vt) {
				return &Catch{
					channel:   chl.Channel,
					topic:     chl.Topic,
					catchType: t.catchType,
					config:    t.config,
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
	if host := t.config.SMTP.Host; host != "" {
		if port := t.config.SMTP.Port; port != "" {
			if user := t.config.SMTP.User; user != "" {
				if password := t.config.SMTP.Password; password != "" {
					nerr = email.NewGoEmail(host, cast.ToInt(port), user, password).Send()
				}
			}
		}
	}
	if nerr != nil {
		if t.config.SendGrid.Key == "" {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		nerr = email.NewSendGrid(ctx, t.config.SendGrid.Key).Send()
	}
	if nerr != nil {
		logger.New().Error(nerr)
	}

}

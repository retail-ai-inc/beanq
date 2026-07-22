package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	beanq "github.com/retail-ai-inc/beanq/v4"
	"github.com/retail-ai-inc/beanq/v4/helper/logger"
	"github.com/spf13/viper"
)

const (
	channel = "sequence-queue-channel"
	topic   = "order-topic"
)

type sequenceMessage struct {
	OrderKey string `json:"orderKey"`
	Body     string `json:"body"`
}

var (
	configOnce sync.Once
	bqConfig   beanq.BeanqConfig
)

func initCnf() *beanq.BeanqConfig {
	configOnce.Do(func() {
		envPath := "../"
		if _, file, _, ok := runtime.Caller(0); ok {
			envPath = filepath.Dir(filepath.Dir(file))
		}

		vp := viper.New()
		vp.AddConfigPath(envPath)
		vp.SetConfigType("json")
		vp.SetConfigName("env")

		if err := vp.ReadInConfig(); err != nil {
			log.Fatalf("Unable to open beanq env.json file: %v", err)
		}
		if err := vp.Unmarshal(&bqConfig); err != nil {
			log.Fatalf("Unable to unmarshal the beanq env.json file: %v", err)
		}
	})
	return &bqConfig
}

func main() {
	ctx := context.Background()
	pub := beanq.New(initCnf())

	orderKeys := []string{
		"order01", "order02", "order03", "order04", "order05",
		"order06", "order07", "order08", "order09", "order10",
	}
	messageCounts := map[string]int{
		"order01": 5,
		"order03": 3,
	}
	defaultMessageCount := 1
	maxMessageCount := 5

	published := 0
	for publishOrder := 1; publishOrder <= maxMessageCount; publishOrder++ {
		for _, orderKey := range orderKeys {
			messageCount := defaultMessageCount
			if count, ok := messageCounts[orderKey]; ok {
				messageCount = count
			}
			if publishOrder > messageCount {
				continue
			}

			msg := sequenceMessage{
				OrderKey: orderKey,
				Body:     fmt.Sprintf("%s-message-%02d", orderKey, publishOrder),
			}
			payload, err := json.Marshal(msg)
			if err != nil {
				logger.New().Error(err)
				continue
			}

			id := fmt.Sprintf("%s-message-%02d", orderKey, publishOrder)
			cmd := pub.BQ().WithContext(ctx).SetId(id).PublishSequence(channel, topic, orderKey, payload)
			if err := cmd.Error(); err != nil {
				logger.New().Error(err)
				continue
			}
			published++
			fmt.Printf("published orderKey=%s publishOrder=%02d id=%s\n", orderKey, publishOrder, id)
		}
	}

	fmt.Printf("published %d messages across %d orderKeys at %s\n", published, len(orderKeys), time.Now().Format(time.RFC3339))
}

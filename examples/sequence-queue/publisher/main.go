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
	CustomerId string `json:"customerId"`
	Body       string `json:"body"`
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

	customerIds := []string{
		"customer01", "customer02", "customer03", "customer04", "customer05",
		"customer06", "customer07", "customer08", "customer09", "customer10",
	}
	messageCounts := map[string]int{
		"customer01": 5,
		"customer03": 3,
	}
	defaultMessageCount := 1
	maxMessageCount := 5

	published := 0
	for publishOrder := 1; publishOrder <= maxMessageCount; publishOrder++ {
		for _, customerId := range customerIds {
			messageCount := defaultMessageCount
			if count, ok := messageCounts[customerId]; ok {
				messageCount = count
			}
			if publishOrder > messageCount {
				continue
			}

			msg := sequenceMessage{
				CustomerId: customerId,
				Body:       fmt.Sprintf("%s-message-%02d", customerId, publishOrder),
			}
			payload, err := json.Marshal(msg)
			if err != nil {
				logger.New().Error(err)
				continue
			}

			id := fmt.Sprintf("%s-message-%02d", customerId, publishOrder)
			cmd := pub.BQ().WithContext(ctx).SetId(id).PublishNewSequence(channel, topic, customerId, payload)
			if err := cmd.Error(); err != nil {
				logger.New().Error(err)
				continue
			}
			published++
			fmt.Printf("published customerId=%s publishOrder=%02d id=%s\n", customerId, publishOrder, id)
		}
	}

	fmt.Printf("published %d messages across %d customerIds at %s\n", published, len(customerIds), time.Now().Format(time.RFC3339))
}

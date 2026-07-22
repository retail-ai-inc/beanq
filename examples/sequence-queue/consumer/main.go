package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"sync"

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
	configOnce      sync.Once
	bqConfig        beanq.BeanqConfig
	sequenceMu      sync.Mutex
	handledSequence = map[string]int{}
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
	csm := beanq.New(initCnf())

	_, err := csm.BQ().WithContext(ctx).SubscribeSequence(channel, topic, beanq.DefaultHandle{
		DoHandle: handle,
		DoCancel: func(ctx context.Context, message *beanq.Message) error {
			return nil
		},
		DoError: func(ctx context.Context, err error) {
			logger.New().Error(err)
		},
	})
	if err != nil {
		logger.New().Error(err)
	}

	fmt.Println("sequence queue consumer started. Run the publisher in another terminal, then press Ctrl+C to stop.")
	csm.Wait(ctx)
}

func handle(ctx context.Context, message *beanq.Message) error {
	var payload sequenceMessage
	if err := json.Unmarshal([]byte(message.Payload), &payload); err != nil {
		return err
	}
	if payload.OrderKey == "" {
		payload.OrderKey = message.OrderKey
	}

	sequenceMu.Lock()
	defer sequenceMu.Unlock()
	expected := handledSequence[payload.OrderKey] + 1
	expectedBody := fmt.Sprintf("%s-message-%02d", payload.OrderKey, expected)
	if payload.Body != expectedBody {
		return fmt.Errorf("out of order: orderKey=%s expectedBody=%s gotBody=%s id=%s", payload.OrderKey, expectedBody, payload.Body, message.Id)
	}
	handledSequence[payload.OrderKey] = expected

	fmt.Printf("handled orderKey=%s body=%s id=%s\n", payload.OrderKey, payload.Body, message.Id)

	return nil
}

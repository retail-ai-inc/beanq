package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
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
	configOnce   sync.Once
	bqConfig     beanq.BeanqConfig
	lastMu       sync.Mutex
	handledCount = map[string]int{}
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

	_, err := csm.BQ().WithContext(ctx).ConsumerSequence(channel, topic, beanq.DefaultHandle{
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
	if payload.CustomerId == "" {
		payload.CustomerId = message.CustomerId
	}

	lastMu.Lock()
	expected := handledCount[payload.CustomerId] + 1
	expectedBody := fmt.Sprintf("%s-message-%02d", payload.CustomerId, expected)
	if payload.Body != expectedBody {
		lastMu.Unlock()
		return fmt.Errorf("out of order: customerId=%s expectedBody=%s gotBody=%s id=%s", payload.CustomerId, expectedBody, payload.Body, message.Id)
	}
	handledCount[payload.CustomerId] = expected
	lastMu.Unlock()

	fmt.Printf("handled customerId=%s body=%s id=%s at=%s\n", payload.CustomerId, payload.Body, message.Id, time.Now().Format(time.RFC3339Nano))
	time.Sleep(300 * time.Millisecond)
	if os.Getenv("BEANQ_SEQUENCE_FAIL_BODY") == payload.Body {
		return fmt.Errorf("simulated failure for %s", payload.Body)
	}
	return nil
}

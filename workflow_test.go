//go:build ci
// +build ci

// WARN: Please use `go test -tags ci ./...` instead of running `go test ./...` if you want to test this file.
package beanq

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/retail-ai-inc/beanq/v3/helper/bstatus"
	"github.com/retail-ai-inc/beanq/v3/helper/tool"
	"github.com/spf13/viper"
)

var (
	taskFailedFlag     int32
	rollbackFailedFlag int32
)

var _ = Describe("DO sequential", Ordered, Label("sequential"), func() {
	var config BeanqConfig
	var client *Client
	var muxClient *MuxClient
	var _channel string
	var _topic string
	var _uuid string

	BeforeAll(func() {
		_channel = "normal_test_channel"
		_topic = "normal_test_topic"
		viper.SetConfigFile("env.testing.json")
		if err := viper.ReadInConfig(); err != nil {
			log.Fatal(err)
		}

		if err := viper.Unmarshal(&config); err != nil {
			log.Fatal(err)
		}

		client = New(&config, WithCaptureExceptionOption(func(ctx context.Context, err any) {
			if err != nil {
				log.Fatal(err)
			}
		}), WithRetryConditions(func(data map[string]any, err error) bool {
			switch {
			case errors.Is(err, bstatus.ErrIdempotent):
				return false
			// TODO: add more retry conditions
			// like:
			// case errors.Is(err, bussiness.Error):
			// 	return false
			default:
				return true
			}
		}))

		muxClient = NewMuxClient(GetBrokerDriver[redis.UniversalClient]())

		go func() {
			ctx := context.Background()
			_, err := client.BQ().WithContext(ctx).SubscribeSequential(_channel, _topic, WorkflowHandler(func(ctx context.Context, wf *Workflow) error {
				wf.Init(
					WfOptionRecordErrorHandler(func(err error) {
						if err == nil {
							return
						}
						fmt.Println("error", err)
					}),
					WfOptionMux(
						muxClient.NewMutex("test", WithExpiry(time.Second*10)),
					),
				)
				wf.NewTask("test").OnRollback(func(task Task) error {
					fmt.Println("rollback")
					switch {
					case atomic.CompareAndSwapInt32(&rollbackFailedFlag, 1, 0):
						return errors.New("rollback failed")
					}
					return nil
				}).OnExecute(func(task Task) error {
					fmt.Println("execute")
					switch {
					case atomic.CompareAndSwapInt32(&taskFailedFlag, 1, 0):
						return errors.New("test failed one time")
					case atomic.LoadInt32(&taskFailedFlag) == -1:
						return errors.New("test failed always")
					case atomic.LoadInt32(&taskFailedFlag) == 0:
						return nil
					}
					return nil
				})

				err := wf.OnRollbackResult(func(taskID string, err error) error {
					fmt.Println("rollback", taskID, err)
					return nil
				}).Run()
				return err
			}))
			Expect(err).To(BeNil())

			// Subscribe without workflow handler
			_, err = client.BQ().WithContext(ctx).SubscribeSequential("no_workflow"+_channel, _topic, DefaultHandle{
				DoHandle: func(ctx context.Context, message *Message) error {
					fmt.Println("execute")
					switch {
					case atomic.CompareAndSwapInt32(&taskFailedFlag, 1, 0):
						return errors.New("test failed one time")
					case atomic.LoadInt32(&taskFailedFlag) == -1:
						return errors.New("test failed always")
					case atomic.LoadInt32(&taskFailedFlag) == 0:
						return nil
					}
					return nil
				},
			})

			Expect(err).To(BeNil())
			client.Wait(ctx)
		}()
	})

	When("workflow", func() {
		Context("normal", func() {
			BeforeEach(func(ctx SpecContext) {
				_uuid = uuid.New().String()
			})

			It("send message", func(ctx SpecContext) {
				resp, err := client.BQ().WithContext(ctx).SetId(_uuid).PublishInSequential(_channel, _topic, []byte("normal test message")).WaitingAck()
				Expect(err).To(BeNil())
				Expect(resp).NotTo(BeNil())
				Expect(resp.Status).To(Equal(bstatus.StatusSuccess))
			}, SpecTimeout(time.Second*5))

			It("send message, failed", func(ctx SpecContext) {
				atomic.StoreInt32(&taskFailedFlag, -1)
				resp, err := client.BQ().WithContext(ctx).SetId(_uuid).PublishInSequential(_channel, _topic, []byte("normal test message")).WaitingAck()
				Expect(err).To(BeNil())
				Expect(resp).NotTo(BeNil())
				Expect(resp.Status).To(Equal(bstatus.StatusFailed))
				Expect(resp.Retry).To(Equal(config.JobMaxRetries))
			}, SpecTimeout(time.Second*5))

			It("send message, failed 1 time", func(ctx SpecContext) {
				atomic.StoreInt32(&taskFailedFlag, 1)
				resp, err := client.BQ().WithContext(ctx).SetId(_uuid).PublishInSequential(_channel, _topic, []byte("normal test message")).WaitingAck()
				Expect(err).To(BeNil())
				Expect(resp).NotTo(BeNil())
				Expect(resp.Status).To(Equal(bstatus.StatusSuccess))
				Expect(resp.Retry).To(Equal(1))
			})

			It("send message, failed 1 time, rollback failed", func(ctx SpecContext) {
				atomic.StoreInt32(&taskFailedFlag, 1)
				atomic.StoreInt32(&rollbackFailedFlag, 1)
				resp, err := client.BQ().WithContext(ctx).SetId(_uuid).PublishInSequential(_channel, _topic, []byte("normal test message")).WaitingAck()
				Expect(err).To(BeNil())
				Expect(resp).NotTo(BeNil())
				Expect(resp.Status).To(Equal(bstatus.StatusSuccess))
				Expect(resp.Retry).To(Equal(1))
			})
		})

		Context("duplicate idempotency key", func() {
			BeforeEach(func(ctx SpecContext) {
				_uuid = "c3b8f27e-f03a-4e9e-b31b-06c6c0e1ef24"
				key := tool.MakeStatusKey(config.Redis.Prefix, _channel, _topic, _uuid)
				fmt.Println("key", key)
				driver := GetBrokerDriver[redis.UniversalClient]()
				err := driver.HSet(ctx, key, "id", _uuid).Err()
				Expect(err).To(BeNil())
			})

			It("send message", func(ctx SpecContext) {
				resp, err := client.BQ().WithContext(ctx).SetId(_uuid).PublishInSequential(_channel, _topic, []byte("normal test message")).WaitingAck()
				Expect(err).Should(MatchError(bstatus.ErrIdempotent))
				Expect(resp).To(BeNil())
			}, SpecTimeout(time.Second*5))
		})
	})
	When("without workflow", func() {
		BeforeEach(func(ctx SpecContext) {
			_uuid = uuid.New().String()
		})

		Context("normal", func() {
			It("send message", func(ctx SpecContext) {
				resp, err := client.BQ().WithContext(ctx).SetId(_uuid).PublishInSequential("no_workflow"+_channel, _topic, []byte("normal test message")).WaitingAck()
				Expect(err).To(BeNil())
				Expect(resp).NotTo(BeNil())
				Expect(resp.Status).To(Equal(bstatus.StatusSuccess))
			}, SpecTimeout(time.Second*5))

			It("send message, failed", func(ctx SpecContext) {
				atomic.StoreInt32(&taskFailedFlag, -1)
				resp, err := client.BQ().WithContext(ctx).SetId(_uuid).PublishInSequential(_channel, _topic, []byte("normal test message")).WaitingAck()
				Expect(err).To(BeNil())
				Expect(resp).NotTo(BeNil())
				Expect(resp.Status).To(Equal(bstatus.StatusFailed))
				Expect(resp.Retry).To(Equal(config.JobMaxRetries))
			}, SpecTimeout(time.Second*5))

			It("send message, failed 1 time", func(ctx SpecContext) {
				atomic.StoreInt32(&taskFailedFlag, 1)
				resp, err := client.BQ().WithContext(ctx).SetId(_uuid).PublishInSequential("no_workflow"+_channel, _topic, []byte("normal test message")).WaitingAck()
				Expect(err).To(BeNil())
				Expect(resp).NotTo(BeNil())
				Expect(resp.Status).To(Equal(bstatus.StatusSuccess))
				Expect(resp.Retry).To(Equal(1))
			})
		})

		Context("duplicate idempotency key", func() {
			BeforeEach(func(ctx SpecContext) {
				_uuid = "duplicate-key-test"
				key := tool.MakeStatusKey(config.Redis.Prefix, "no_workflow"+_channel, _topic, _uuid)
				driver := GetBrokerDriver[redis.UniversalClient]()
				err := driver.HSet(ctx, key, "id", _uuid).Err()
				Expect(err).To(BeNil())
			})

			It("send message", func(ctx SpecContext) {
				resp, err := client.BQ().WithContext(ctx).SetId(_uuid).PublishInSequential("no_workflow"+_channel, _topic, []byte("duplicate test message")).WaitingAck()
				Expect(err).Should(MatchError(bstatus.ErrIdempotent))
				Expect(resp).To(BeNil())
			}, SpecTimeout(time.Second*5))
		})
	})
})

//go:build ci
// +build ci

// WARN: Please use `go test -tags ci ./...` instead of running `go test ./...` if you want to test this file.
package beanq

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
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
	config    BeanqConfig
	client    *Client
	muxClient *MuxClient
	_channel  string
	_topic    string
	_uuid     string
	once      sync.Once
)

const (
	normalMessage         = "normal message"
	failedOneTimeMessage  = "failed one time message"
	alwaysFailedMessage   = "always failed message"
	rollbackFailedMessage = "rollback failed message"
)

var _ = BeforeSuite(func() {
	once.Do(
		func() {
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
				_, err := client.BQ().WithContext(ctx).SubscribeSequential(_channel, _topic, func() WorkflowHandler {
					var retryCount int

					return func(ctx context.Context, wf *Workflow) error {
						message := wf.Message()
						payload := message.Payload

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

							switch payload {
							case rollbackFailedMessage:
								return errors.New("rollback failed")
							}
							return nil
						}).OnExecute(func(task Task) error {
							fmt.Println("execute")
							switch payload {
							case normalMessage:
								return nil
							case alwaysFailedMessage:
								return errors.New(alwaysFailedMessage)
							case failedOneTimeMessage, rollbackFailedMessage:
								if retryCount > 0 {
									retryCount = 0
									return nil
								}
								retryCount++
								return errors.New(payload)
							}

							return nil
						})

						err := wf.OnRollbackResult(func(taskID string, err error) error {
							fmt.Println("rollback", taskID, err)
							return nil
						}).Run()
						return err
					}
				}())
				Expect(err).To(BeNil())

				// Subscribe without workflow handler
				_, err = client.BQ().WithContext(ctx).SubscribeSequential("no_workflow"+_channel, _topic, func() DefaultHandle {
					var retryCount int

					return DefaultHandle{
						DoHandle: func(ctx context.Context, message *Message) error {
							fmt.Println("execute")
							payload := message.Payload
							switch payload {
							case failedOneTimeMessage:
								if retryCount > 0 {
									return nil
								}
								retryCount++
								return errors.New(failedOneTimeMessage)
							case alwaysFailedMessage:
								return errors.New(alwaysFailedMessage)
							case normalMessage:
								return nil
							}
							return nil
						},
					}
				}())

				Expect(err).To(BeNil())
				client.Wait(ctx)
			}()
		},
	)
})

var _ = Describe("DO sequential Ordered", Label("sequential"), func() {
	When("workflow", Ordered, func() {
		Context("normal", func() {
			BeforeEach(func(ctx SpecContext) {
				_uuid = uuid.New().String()
			})

			It("send message", func(ctx SpecContext) {
				resp, err := client.BQ().WithContext(ctx).SetId(_uuid).PublishInSequential(_channel, _topic, []byte(normalMessage)).WaitingAck()
				Expect(err).To(BeNil())
				Expect(resp).NotTo(BeNil())
				Expect(resp.Status).To(Equal(bstatus.StatusSuccess))
			}, SpecTimeout(time.Second*5))

			It("send message, failed", func(ctx SpecContext) {
				resp, err := client.BQ().WithContext(ctx).SetId(_uuid).PublishInSequential(_channel, _topic, []byte(alwaysFailedMessage)).WaitingAck()
				Expect(err).To(BeNil())
				Expect(resp).NotTo(BeNil())
				Expect(resp.Status).To(Equal(bstatus.StatusFailed))
				Expect(resp.Retry).To(Equal(config.JobMaxRetries))
			}, SpecTimeout(time.Second*5))

			It("send message, failed 1 time", func(ctx SpecContext) {
				resp, err := client.BQ().WithContext(ctx).SetId(_uuid).PublishInSequential(_channel, _topic, []byte(failedOneTimeMessage)).WaitingAck()
				Expect(err).To(BeNil())
				Expect(resp).NotTo(BeNil())
				Expect(resp.Status).To(Equal(bstatus.StatusSuccess))
				Expect(resp.Retry).To(Equal(1))
			})

			It("send message, failed 1 time, rollback failed", func(ctx SpecContext) {
				resp, err := client.BQ().WithContext(ctx).SetId(_uuid).PublishInSequential(_channel, _topic, []byte(rollbackFailedMessage)).WaitingAck()
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
				resp, err := client.BQ().WithContext(ctx).SetId(_uuid).PublishInSequential(_channel, _topic, []byte(normalMessage)).WaitingAck()
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
				resp, err := client.BQ().WithContext(ctx).SetId(_uuid).PublishInSequential("no_workflow"+_channel, _topic, []byte(normalMessage)).WaitingAck()
				Expect(err).To(BeNil())
				Expect(resp).NotTo(BeNil())
				Expect(resp.Status).To(Equal(bstatus.StatusSuccess))
			}, SpecTimeout(time.Second*5))

			It("send message, failed", func(ctx SpecContext) {
				resp, err := client.BQ().WithContext(ctx).SetId(_uuid).PublishInSequential(_channel, _topic, []byte(alwaysFailedMessage)).WaitingAck()
				Expect(err).To(BeNil())
				Expect(resp).NotTo(BeNil())
				Expect(resp.Status).To(Equal(bstatus.StatusFailed))
				Expect(resp.Retry).To(Equal(config.JobMaxRetries))
			}, SpecTimeout(time.Second*5))

			It("send message, failed 1 time", func(ctx SpecContext) {
				resp, err := client.BQ().WithContext(ctx).SetId(_uuid).PublishInSequential("no_workflow"+_channel, _topic, []byte(failedOneTimeMessage)).WaitingAck()
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
				resp, err := client.BQ().WithContext(ctx).SetId(_uuid).PublishInSequential("no_workflow"+_channel, _topic, []byte(normalMessage)).WaitingAck()
				Expect(err).Should(MatchError(bstatus.ErrIdempotent))
				Expect(resp).To(BeNil())
			}, SpecTimeout(time.Second*5))
		})
	})
})

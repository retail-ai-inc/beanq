package beanq

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/retail-ai-inc/beanq/v3/helper/bstatus"
	"github.com/retail-ai-inc/beanq/v3/helper/tool"
	"github.com/spf13/viper"
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
		}))

		muxClient = NewMuxClient(GetBrokerDriver[redis.UniversalClient]())

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
				return nil
			}).OnExecute(func(task Task) error {
				fmt.Println("execute")
				return nil
			})

			err := wf.OnRollbackResult(func(taskID string, err error) error {
				fmt.Println("rollback", taskID, err)
				return nil
			}).Run()
			Expect(err).To(BeNil())
			return nil
		}))
		Expect(err).To(BeNil())

		_, err = client.BQ().WithContext(ctx).SubscribeSequential(_channel+"_failed", _topic+"_failed", WorkflowHandler(func(ctx context.Context, wf *Workflow) error {
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
				return nil
			}).OnExecute(func(task Task) error {
				fmt.Println("execute")
				return errors.New("test failed")
			})

			err := wf.OnRollbackResult(func(taskID string, err error) error {
				fmt.Println("rollback", taskID, err)
				return nil
			}).Run()
			Expect(err).NotTo(BeNil())
			return err
		}))
		Expect(err).To(BeNil())

		go func() {
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

			It("send failed message", func(ctx SpecContext) {
				resp, err := client.BQ().WithContext(ctx).SetId(_uuid).PublishInSequential(_channel+"_failed", _topic+"_failed", []byte("normal test message")).WaitingAck()
				Expect(err).To(BeNil())
				Expect(resp).NotTo(BeNil())
				Expect(resp.Status).To(Equal(bstatus.StatusFailed))
			}, SpecTimeout(time.Second*5))

			It("send parallel message", func(ctx SpecContext) {
				for i := 0; i < 10; i++ {
					go func() {
						resp, err := client.BQ().WithContext(ctx).SetId(_uuid).PublishInSequential(_channel, _topic, []byte("normal test message")).WaitingAck()
						Expect(err).To(BeNil())
						Expect(resp).NotTo(BeNil())
						Expect(resp.Status).To(Equal(bstatus.StatusSuccess))
					}()
				}
			}, SpecTimeout(time.Second*5))
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
			muxClient = NewMuxClient(GetBrokerDriver[redis.UniversalClient]())
			_uuid = uuid.New().String()

			// Subscribe without workflow handler
			_, err := client.BQ().WithContext(ctx).SubscribeSequential(_channel, "no_workflow_topic", DefaultHandle{
				DoHandle: func(ctx context.Context, message *Message) error {
					fmt.Println("msg", message)
					return nil
				},
			})
			Expect(err).To(BeNil())
		})

		Context("normal", func() {
			It("send and receive message", func(ctx SpecContext) {
				resp, err := client.BQ().WithContext(ctx).SetId(_uuid).PublishInSequential(_channel, _topic, []byte("normal test message")).WaitingAck()
				Expect(err).To(BeNil())
				Expect(resp).NotTo(BeNil())
				Expect(resp.Status).To(Equal(bstatus.StatusSuccess))
			}, SpecTimeout(time.Second*5))
		})

		Context("duplicate idempotency key", func() {
			BeforeEach(func(ctx SpecContext) {
				_uuid = "duplicate-key-test"
				key := tool.MakeStatusKey(config.Redis.Prefix, _channel, "no_workflow_topic", _uuid)
				driver := GetBrokerDriver[redis.UniversalClient]()
				err := driver.HSet(ctx, key, "id", _uuid).Err()
				Expect(err).To(BeNil())
			})

			It("send and receive message", func(ctx SpecContext) {
				resp, err := client.BQ().WithContext(ctx).SetId(_uuid).PublishInSequential(_channel, "no_workflow_topic", []byte("duplicate test message")).WaitingAck()
				Expect(err).Should(MatchError(bstatus.ErrIdempotent))
				Expect(resp).To(BeNil())
			}, SpecTimeout(time.Second*5))
		})
	})
})

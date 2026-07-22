/*
# Normal Consumer

Consuming messages from Beanq can be done by creating an instance of a Consumer and supplying it a handler.

	package main

	import (
		"context"
		"time"

		"github.com/retail-ai-inc/beanq/v4"
		"github.com/retail-ai-inc/beanq/v4/helper/logger"
	)

	func main() {

		channel := "default-channel"
		topic := "default-topic"

		config, err := beanq.NewConfig("./", "json", "env")
		if err != nil {
			logger.New().Error(err)
			return
		}
		consumer := beanq.New(config)

		// register delay consumer
		ctx := context.Background()
		_, err = consumer.BQ().WithContext(ctx).Subscribe(channel, topic, beanq.DefaultHandle{
			DoHandle: func(ctx context.Context, message *beanq.Message) error {
				logger.New().With("default-channel", "default-topic").Info(message.Payload)
				return nil
			},
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
		// begin to consume information
		consumer.Wait(ctx)
	}

# Normal Producer

Normal queues use fixed Redis Stream partitions. Messages are assigned by a
stable hash of the Beanq message ID, while all partitions share the configured
consumer worker pool. Configure normalQueuePartitions consistently across
publishers and consumers. The maxLen setting applies to each partition.

Delay queues reuse normalQueuePartitions. PublishAtTime hashes the Beanq message
ID to a partition whose scheduled ZSET and ready Stream share one Redis Cluster
hash slot; different partitions use different slots. The maxLen setting applies
to each ready Stream partition.

The partitioned v2 layout does not consume messages left in the legacy single
normal Stream or delay ZSET/Stream. Drain or remove legacy messages before
upgrading.

Producing messages can be done by creating an instance of a Producer.

	package main

	import (
		"context"

		"github.com/retail-ai-inc/beanq/v4"
		"github.com/retail-ai-inc/beanq/v4/helper/json"
		"github.com/retail-ai-inc/beanq/v4/helper/logger"
	)

	func main() {

		channel := "default-channel"
		topic := "default-topic"

		config, err := beanq.NewConfig("./", "json", "env")
		if err != nil {
			logger.New().Error(err)
			return
		}
		pub := beanq.New(config)

		b := []byte(`{"msg":"publish testing"}`)

		if err := pub.BQ().WithContext(context.Background()).Publish(channel, topic, b); err != nil {
			logger.New().Error(err)
		}

	}

# Delay Consumer

Use the SubscribeToDelay function to consume delayed messages

	package main

	import (
		"context"

		"github.com/retail-ai-inc/beanq/v4"
		"github.com/retail-ai-inc/beanq/v4/helper/logger"
	)

	func main() {

		channel := "delay-channel"
		topic := "delay-topic"

		config, err := beanq.NewConfig("./", "json", "env")
		if err != nil {
			logger.New().Error(err)
			return
		}
		ctx := context.Background()
		csm := beanq.New(config)

		// register delay consumer
		_, err = csm.BQ().WithContext(ctx).SubscribeToDelay(channel, topic, beanq.DefaultHandle{
			DoHandle: func(ctx context.Context, message *beanq.Message) error {
				logger.New().With("delay-channel", "delay-topic").Info(message.Payload)
				return nil
			},
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

		csm.Wait(ctx)

	}

# Delay Publisher

Publish delayed messages, taking execution time and priority as examples

	package main

	import (
		"context"
		"time"

		"github.com/retail-ai-inc/beanq/v4"
		"github.com/retail-ai-inc/beanq/v4/helper/json"
		"github.com/retail-ai-inc/beanq/v4/helper/logger"
		"github.com/spf13/cast"
	)

	func main() {
		config, err := beanq.NewConfig("./", "json", "env")
		if err != nil {
			logger.New().Error(err)
			return
		}
		pub := beanq.New(config)

		m := make(map[string]any)
		ctx := context.Background()
		now := time.Now()

		//Sort by execution time, the smaller the priority, the earlier it is consumed.
		//For messages of the same time, the larger the priority, the earlier it is consumed.
		for i := 0; i < 10; i++ {

			delayT := now
			y := 0
			m["delayMsg"] = "new msg" + cast.ToString(i)

			b, _ := json.Marshal(m)

			if i == 4 {
				//setting priority
				y = 8
			}
			if i == 3 {
				// delay execution time
				delayT = now.Add(10 * time.Second)
			}

			if err := pub.BQ().WithContext(ctx).Priority(float64(y)).PublishAtTime("delay-channel", "order-topic", b, delayT); err != nil {
				logger.New().Error(err)
			}
		}
	}

# Keyed Sequence Queue

Keyed Sequence Queue hashes each orderKey into a fixed scheduler partition. Messages for one order key are finalized in FIFO order by one valid owner, while different order keys may be consumed concurrently. Delivery is at-least-once, so handlers should be idempotent by message ID.

Publish with PublishNewSequence(channel, topic, orderKey, payload) and subscribe with ConsumerSequence. Configure sequenceQueuePartitions consistently across publishers and consumers; once queue metadata exists, the partition count cannot change in place.

# Work Flow

How Workflow Works

	package main

	import (
		"context"
		"fmt"
		"log"
		"time"

		"github.com/retail-ai-inc/beanq/v4"
		"github.com/retail-ai-inc/beanq/v4/helper/logger"
	)

	var index int = 3

	func main() {

		ctx := context.Background()

		config, err := beanq.NewConfig("./", "json", "env")
		if err != nil {
			logger.New().Error(err)
			return
		}
		csm := beanq.New(config)

		_, berr := csm.BQ().WithContext(ctx).SubscribeSequence("delay-channel", "order-topic", beanq.WorkflowHandler(func(ctx context.Context, wf *beanq.Workflow) error {
			index++
			fmt.Println("index:", index)
			wf.NewTask().OnRollback(func(task beanq.Task) error {
				if index%3 == 0 {
					return fmt.Errorf("rollback error:%d", index)
				} else if index%4 == 0 {
					panic("rollback panic test")
				}
				log.Println(task.ID()+" rollback-1:", wf.Message().Id)
				return nil
			}).OnExecute(func(task beanq.Task) error {
				log.Println(task.ID() + " job-1")
				time.Sleep(time.Second * 2)
				return nil
			})

			wf.NewTask().OnRollback(func(task beanq.Task) error {
				log.Println(task.ID()+" rollback-2:", wf.Message().Id)
				return nil
			}).OnExecute(func(task beanq.Task) error {
				log.Println(task.ID() + " job-2")
				time.Sleep(time.Second * 1)
				return nil
			})

			wf.NewTask().OnRollback(func(task beanq.Task) error {
				log.Println(task.ID()+" rollback-3:", wf.Message().Id)
				return nil
			}).OnExecute(func(task beanq.Task) error {
				if index%2 == 0 {
					return fmt.Errorf("execute error: %d", index)
				} else if index == 7 {
					panic("execute panic test")
				}
				log.Println(task.ID() + " job-3")
				time.Sleep(time.Second * 1)
				return nil
			})

			berr := wf.OnRollbackResult(func(taskID string, berr error) {
				if berr == nil {
					return
				}
				log.Printf("%s rollback error: %v\n", taskID, berr)
			}).Run()
			if berr != nil {
				return berr
			}
			return nil
		}))

		if berr != nil {
			logger.New().Error(berr)
		}
	}

# Start UI

The client actively enables the monitoring platform UI

	package main

	import (
		"context"
		"log"

		"github.com/retail-ai-inc/beanq/v4"
	)

	func main() {
		config, err := beanq.NewConfig("./", "json", "env")
		if err != nil {
			log.Fatalf("Unable to open beanq env.json file: %v", err)

		}
		csm := beanq.New(config)
		if err := csm.ServeHTTP(context.Background()); err != nil {
			panic(err)
		}
	}
*/
package beanq

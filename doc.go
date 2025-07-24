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

# Sequence Queue

It is equivalent to the key of the event, which allows only one event to be consumed at the same time.

	package main

	import (
		"context"
		"encoding/json"
		"log"
		"time"

		"github.com/retail-ai-inc/beanq/v4"
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

		for i := 0; i < 3; i++ {
			id := cast.ToString(i)

			m := make(map[string]any)
			m["delayMsg"] = "new msg" + id

			b, _ := json.Marshal(m)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
			defer cancel()
			// At the same time, for the same orderKey: aa,
			// only one consumption is allowed, and the others will be blocked until SetLockOrderKeyTTL(10*time.Second) expires and deletes the orderKey
			result, err := pub.BQ().WithContext(ctx).
				SetId(id).SetLockOrderKeyTTL(10*time.Second).
				PublishInSequenceByLock("delay-channel", "order-topic", "aa", b).WaitingAck()
			if err != nil {
				logger.New().Error(err, m)
			} else {
				log.Printf("ID:%+v \n", result.Id)
			}
		}

	}

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

		_, berr := csm.BQ().WithContext(ctx).SubscribeToSequence("delay-channel", "order-topic", beanq.WorkflowHandler(func(ctx context.Context, wf *beanq.Workflow) error {
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
		csm.ServeHttp(context.Background())
	}
*/
package beanq

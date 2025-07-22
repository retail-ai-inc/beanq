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

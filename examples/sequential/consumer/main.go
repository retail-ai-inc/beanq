package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/retail-ai-inc/beanq/v4/helper/logger"

	"github.com/retail-ai-inc/beanq/v4"
)

//nolint:unused
type seqCustomer struct {
	metadata string
}

//nolint:unused
func (t *seqCustomer) Handle(ctx context.Context, message *beanq.Message) error {
	time.Sleep(time.Second * 4)
	log.Printf("%s:%v\n", t.metadata, message)
	return nil
}

//nolint:unused
func (t *seqCustomer) Cancel(ctx context.Context, message *beanq.Message) error {
	return nil
}

//nolint:unused
func (t *seqCustomer) Error(ctx context.Context, err error) {
}

var ErrorSkip = errors.New("SKIP ERROR")

func main() {

	config, err := beanq.NewConfig("./", "json", "env")
	if err != nil {
		logger.New().Error(err)
		return
	}
	csm := beanq.New(config)
	beanq.InitWorkflow(&config.Redis, &config.Workflow)

	ctx := context.Background()
	_, berr := csm.BQ().WithContext(ctx).SubscribeToSequence("sequential-channel", "order-topic", beanq.WorkflowHandler(func(ctx context.Context, wf *beanq.Workflow) error {
		index, err := strconv.Atoi(wf.GetGid())
		if err != nil {
			return err
		}

		wf.Init(beanq.WfSkipper(func(err error) bool {
			if err == nil {
				return true
			}
			if errors.Is(err, ErrorSkip) {
				return true
			}
			return false
		}))

		wf.NewTask().Skipper(func(err error) bool {
			if err == nil {
				return true
			}
			if errors.Is(err, ErrorSkip) {
				return true
			}
			return false
		}).OnRollback(func(task beanq.Task) error {
			if index%3 == 0 {
				return fmt.Errorf("rollback error:%d", index)
			} else if index%4 == 0 {
				panic("rollback panic test")
			} else if index%5 == 0 {
				return fmt.Errorf("rollback error:%w", ErrorSkip)
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
				return fmt.Errorf("execute error: %s %d", wf.GetGid(), index)
			} else if index%3 == 0 {
				return fmt.Errorf("execute error:%w", ErrorSkip)
			} else if index%7 == 0 {
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

	// begin to consume information
	csm.Wait(ctx)
}

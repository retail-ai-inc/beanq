package main

import (
	"context"
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

var (
	configOnce sync.Once
	bqConfig   beanq.BeanqConfig
	index      = 3
)

func initCnf() *beanq.BeanqConfig {
	configOnce.Do(func() {
		envPath := "./"
		if _, file, _, ok := runtime.Caller(0); ok {
			envPath = filepath.Dir(file)
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

	_, err := csm.BQ().WithContext(ctx).SubscribeSequence("delay-channel", "order-topic", beanq.WorkflowHandler(func(ctx context.Context, workflow *beanq.Workflow) error {
		index++
		fmt.Println("index:", index)
		workflow.NewTask().OnRollback(func(task beanq.Task) error {
			if index%3 == 0 {
				return fmt.Errorf("rollback error:%d", index)
			} else if index%4 == 0 {
				panic("rollback panic test")
			}
			log.Println(task.ID()+" rollback-1:", workflow.Message().Id)
			return nil
		}).OnExecute(func(task beanq.Task) error {
			log.Println(task.ID() + " job-1")
			time.Sleep(2 * time.Second)
			return nil
		})

		workflow.NewTask().OnRollback(func(task beanq.Task) error {
			log.Println(task.ID()+" rollback-2:", workflow.Message().Id)
			return nil
		}).OnExecute(func(task beanq.Task) error {
			log.Println(task.ID() + " job-2")
			time.Sleep(time.Second)
			return nil
		})

		workflow.NewTask().OnRollback(func(task beanq.Task) error {
			log.Println(task.ID()+" rollback-3:", workflow.Message().Id)
			return nil
		}).OnExecute(func(task beanq.Task) error {
			if index%2 == 0 {
				return fmt.Errorf("execute error: %d", index)
			} else if index == 7 {
				panic("execute panic test")
			}
			log.Println(task.ID() + " job-3")
			time.Sleep(time.Second)
			return nil
		})

		return workflow.OnRollbackResult(func(taskID string, rollbackErr error) {
			if rollbackErr != nil {
				log.Printf("%s rollback error: %v\n", taskID, rollbackErr)
			}
		}).Run()
	}))

	if err != nil {
		logger.New().Error(err)
	}
}

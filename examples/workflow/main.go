package main

import (
	"fmt"
	"github.com/retail-ai-inc/beanq/v3"
	"github.com/retail-ai-inc/beanq/v3/helper/logger"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
	"log"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

var (
	configOnce sync.Once
	bqConfig   beanq.BeanqConfig
)

func initCnf() *beanq.BeanqConfig {
	configOnce.Do(func() {
		var envPath string = "./"
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

		// IMPORTANT: Unmarshal the env.json into global Config object.
		if err := vp.Unmarshal(&bqConfig); err != nil {
			log.Fatalf("Unable to unmarshal the beanq env.json file: %v", err)
		}
	})
	return &bqConfig
}

var index int = 3

func main() {

	ctx := context.Background()
	config := initCnf()
	csm := beanq.New(config)

	_, berr := csm.BQ().WithContext(ctx).SubscribeSequential("delay-channel", "order-topic", beanq.WorkflowHandler(func(ctx context.Context, wf *beanq.Workflow) error {
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

		berr := wf.OnRollbackResult(func(taskID string, berr error) error {
			if berr == nil {
				return nil
			}
			log.Printf("%s rollback error: %v\n", taskID, berr)
			return nil
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

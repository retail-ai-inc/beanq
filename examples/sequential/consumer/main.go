package main

import (
	"context"
	"fmt"
	"log"
	_ "net/http/pprof"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/retail-ai-inc/beanq/v3/helper/logger"

	beanq "github.com/retail-ai-inc/beanq/v3"
	"github.com/spf13/viper"
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

type seqCustomer struct {
	metadata string
}

func (t *seqCustomer) Handle(ctx context.Context, message *beanq.Message) error {
	time.Sleep(time.Second * 4)
	log.Printf("%s:%v\n", t.metadata, message)
	return nil
}

func (t *seqCustomer) Cancel(ctx context.Context, message *beanq.Message) error {
	return nil
}

func (t *seqCustomer) Error(ctx context.Context, err error) {
}

var index int32

func main() {
	config := initCnf()
	csm := beanq.New(config)
	beanq.InitWorkflow(&config.Redis)

	ctx := context.Background()
	_, berr := csm.BQ().WithContext(ctx).SubscribeSequential("delay-channel", "order-topic", beanq.WorkflowHandler(func(ctx context.Context, wf *beanq.Workflow) error {
		atomic.AddInt32(&index, 1)
		fmt.Println("index:", atomic.LoadInt32(&index))
		wf.NewTask().OnRollback(func(task beanq.Task) error {
			if atomic.LoadInt32(&index)%3 == 0 {
				return fmt.Errorf("rollback error:%d", atomic.LoadInt32(&index))
			} else if atomic.LoadInt32(&index)%4 == 0 {
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
			if atomic.LoadInt32(&index)%2 == 0 {
				return fmt.Errorf("execute error: %d", atomic.LoadInt32(&index))
			} else if atomic.LoadInt32(&index) == 7 {
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
			return
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

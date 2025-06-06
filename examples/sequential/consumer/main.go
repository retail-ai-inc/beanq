package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	_ "net/http/pprof"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
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

var SkipError = errors.New("SKIP ERROR")

func main() {
	config := initCnf()
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
			if errors.Is(err, SkipError) {
				return true
			}
			return false
		}))

		wf.NewTask().Skipper(func(err error) bool {
			if err == nil {
				return true
			}
			if errors.Is(err, SkipError) {
				return true
			}
			return false
		}).OnRollback(func(task beanq.Task) error {
			if index%3 == 0 {
				return fmt.Errorf("rollback error:%d", index)
			} else if index%4 == 0 {
				panic("rollback panic test")
			} else if index%5 == 0 {
				return fmt.Errorf("rollback error:%w", SkipError)
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
				return fmt.Errorf("execute error:%w", SkipError)
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

package main

import (
	"context"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/retail-ai-inc/beanq"
	"github.com/retail-ai-inc/beanq/helper/logger"
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

var index atomic.Int32

func main() {
	config := initCnf()
	csm := beanq.New(config)
	go func() {
		log.Println(http.ListenAndServe(":6060", nil))
	}()
	ctx := context.Background()
	_, err := csm.BQ().WithContext(ctx).Dynamic().SubscribeSequential("delay-channel", "*", beanq.DefaultHandle{
		DoHandle: func(ctx context.Context, message *beanq.Message) error {
			time.Sleep(time.Second * time.Duration(rand.Int63n(5)))
			logger.New().Info("default handler ", message.Id, message.Topic)
			return nil
		},
		DoCancel: func(ctx context.Context, message *beanq.Message) error {
			logger.New().Info("default cancel ", message.Id)
			return nil
		},
		DoError: func(ctx context.Context, err error) {
			logger.New().Info("default error ", err)
		},
	})
	if err != nil {
		logger.New().Error(err)
	}

	/*_, err := csm.BQ().WithContext(ctx).Dynamic().SubscribeSequential("delay-channel", "*", beanq.WorkflowHandler(func(ctx context.Context, wf *beanq.Workflow) error {
		index.Add(1)
		wf.NewTask().OnRollback(func(task beanq.Task) error {
			if index.Load()%3 == 0 {
				return fmt.Errorf("rollback error:%d", index.Load())
			} else if index.Load()%4 == 0 {
				panic("rollback panic test")
			}
			logger.New().Info("topic:", wf.Message().Topic, task.ID()+" rollback-1:", wf.Message().Id)
			return nil
		}).OnExecute(func(task beanq.Task) error {
			logger.New().Info("topic:", wf.Message().Topic, task.ID()+"execute job-1")
			time.Sleep(time.Second * 2)
			return nil
		})

		wf.NewTask().OnRollback(func(task beanq.Task) error {
			logger.New().Info("topic:", wf.Message().Topic, task.ID()+" rollback-2:", wf.Message().Id)
			return nil
		}).OnExecute(func(task beanq.Task) error {
			logger.New().Info("topic:", wf.Message().Topic, task.ID()+"execute job-2")
			time.Sleep(time.Second * 1)
			return nil
		})

		wf.NewTask().OnRollback(func(task beanq.Task) error {
			logger.New().Info("topic:", wf.Message().Topic, task.ID()+" rollback-3:", wf.Message().Id)
			return nil
		}).OnExecute(func(task beanq.Task) error {
			if index.Load()%2 == 0 {
				return fmt.Errorf("execute error: %d", index.Load())
			} else if index.Load() == 7 {
				panic("execute panic test")
			}
			logger.New().Info("topic:", wf.Message().Topic, task.ID()+"execute job-3")
			time.Sleep(time.Second * 1)
			return nil
		})

		err := wf.WithRollbackResultHandler(func(taskID string, err error) {
			if err == nil {
				return
			}
			logger.New().Info("topic:", wf.Message().Topic, taskID, "rollback error: ", err)
		}).Run()
		if err != nil {
			return err
		}
		return nil
	}))
	if err != nil {
		logger.New().Error(err)
	}*/
}

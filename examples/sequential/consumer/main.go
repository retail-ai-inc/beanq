package main

import (
	"context"
	"log"
	_ "net/http/pprof"
	"path/filepath"
	"runtime"
	"sync"
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

var index int

func main() {
	config := initCnf()
	csm := beanq.New(config)

	ctx := context.Background()
	// _, err := csm.BQ().WithContext(ctx).SubscribeSequential("delay-channel", "order-topic", beanq.WorkflowHandler(func(ctx context.Context, wf *beanq.Workflow) error {
	// 	index++
	// 	fmt.Println("index:", index)
	// 	wf.NewTask().OnRollback(func(task beanq.Task) error {
	// 		if index%3 == 0 {
	// 			return fmt.Errorf("rollback error:%d", index)
	// 		} else if index%4 == 0 {
	// 			panic("rollback panic test")
	// 		}
	// 		log.Println(task.ID()+" rollback-1:", wf.Message().Id)
	// 		return nil
	// 	}).OnExecute(func(task beanq.Task) error {
	// 		log.Println(task.ID() + " job-1")
	// 		time.Sleep(time.Second * 2)
	// 		return nil
	// 	})
	//
	// 	wf.NewTask().OnRollback(func(task beanq.Task) error {
	// 		log.Println(task.ID()+" rollback-2:", wf.Message().Id)
	// 		return nil
	// 	}).OnExecute(func(task beanq.Task) error {
	// 		log.Println(task.ID() + " job-2")
	// 		time.Sleep(time.Second * 1)
	// 		return nil
	// 	})
	//
	// 	wf.NewTask().OnRollback(func(task beanq.Task) error {
	// 		log.Println(task.ID()+" rollback-3:", wf.Message().Id)
	// 		return nil
	// 	}).OnExecute(func(task beanq.Task) error {
	// 		if index%2 == 0 {
	// 			return fmt.Errorf("execute error: %d", index)
	// 		} else if index == 7 {
	// 			panic("execute panic test")
	// 		}
	// 		log.Println(task.ID() + " job-3")
	// 		time.Sleep(time.Second * 1)
	// 		return nil
	// 	})
	//
	// 	err := wf.OnRollbackResult(func(taskID string, err error) error {
	// 		if err == nil {
	// 			return nil
	// 		}
	// 		log.Printf("%s rollback error: %v\n", taskID, err)
	// 		return nil
	// 	}).Run()
	// 	if err != nil {
	// 		return err
	// 	}
	// 	return nil
	// }))
	// if err != nil {
	// 	logger.New().Error(err)
	// }

	_, err := csm.BQ().WithContext(ctx).SubscribeSequential("delay-channel", "order-topic", beanq.DefaultHandle{
		DoHandle: func(ctx context.Context, message *beanq.Message) error {
			// message.Response = fmt.Sprintf("test val,id=%+v", message.Id)
			// log.Println("default handler ", message.Id)
			return nil
		},
		DoCancel: func(ctx context.Context, message *beanq.Message) error {
			log.Println("default cancel ", message.Id)
			return beanq.NilCancel
		},
		DoError: func(ctx context.Context, err error) {
			log.Println("default error ", err)
		},
	})
	if err != nil {
		logger.New().Error(err)
	}
	// go func() {
	//	for {
	//		time.Sleep(3 * time.Second)
	//		fmt.Println(runtime.NumGoroutine())
	//	}
	// }()
	// _, err = csm.BQ().WithContext(ctx).SubscribeSequential("delay-channel", "order-topic", &seqCustomer{
	// 	metadata: "I am a custom",
	// })
	// if err != nil {
	// 	logger.New().Error(err)
	// }
	// begin to consume information
	csm.Wait(ctx)

}

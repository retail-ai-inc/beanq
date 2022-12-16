package beanq

import (
	"beanq/client"
	"beanq/helper/json"
	"beanq/task"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/spf13/cast"
	"log"
	"testing"
	"time"
)

var (
	queue    = "ch2"
	group    = "g2"
	consumer = "cs1"
	options  task.Options
	clt      Beanq
)

func init() {
	options = task.Options{
		RedisOptions: &redis.Options{
			Addr:      Env.Queue.Redis.Host + ":" + cast.ToString(Env.Queue.Redis.Port),
			Dialer:    nil,
			OnConnect: nil,
			Username:  "",
			Password:  Env.Queue.Redis.Password,
			DB:        Env.Queue.Redis.Db,
		},
		KeepJobInQueue:           Env.Queue.KeepJobsInQueue,
		KeepFailedJobsInHistory:  Env.Queue.KeepFailedJobsInHistory,
		KeepSuccessJobsInHistory: Env.Queue.KeepSuccessJobsInHistory,
		MinWorkers:               Env.Queue.MinWorkers,
		JobMaxRetry:              Env.Queue.JobMaxRetries,
		Prefix:                   Env.Queue.Redis.Prefix,
	}
	clt = NewBeanq("redis", options)
}
func TestPublishOne(t *testing.T) {

	m := make(map[int]string)
	m[0] = "k----" + cast.ToString(0)

	d, _ := json.Marshal(m)
	task := task.NewTask("", d)
	cmd, err := clt.Publish(task, client.Queue("ch2"))
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Printf("%+v \n", cmd)

	defer clt.Close()
}

/*
  - TestPublish
  - @Description:
    publisher
  - @param t
*/
func TestPublish1(t *testing.T) {

	for i := 0; i < 5; i++ {
		m := make(map[int]string)
		m[i] = "k----" + cast.ToString(i)

		d, _ := json.Marshal(m)
		task := task.NewTask("", d)
		cmd, err := clt.Publish(task, client.Queue("ch2"))
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Printf("%+v \n", cmd)
	}
	defer clt.Close()
}
func TestDelayPublish(t *testing.T) {
	m := make(map[string]string)
	m["delayMsg"] = "new msg11111"
	b, _ := json.Marshal(&m)
	task := task.NewTask("update", b)

	delayT := time.Now().Add(60 * time.Second)
	_, err := clt.DelayPublish(task, delayT, client.Queue("delay-ch"))
	if err != nil {
		t.Fatal(err.Error())
	}
	defer clt.Close()
}
func TestRetry(t *testing.T) {

	err := retry(func() error {
		fmt.Println("function body")
		return errors.New("错误")
		//return nil
	}, 500*time.Millisecond)

	fmt.Println(err)

}

func retry(f func() error, delayTime time.Duration) error {
	retryFlag := make(chan error)
	stopRetry := make(chan bool, 1)

	go func(duration time.Duration, errChan chan error, stop chan bool) {
		index := 1
		count := 3

		for {
			go time.AfterFunc(duration, func() {
				errChan <- f()
			})
			err := <-errChan
			if err == nil {
				stop <- true
				close(errChan)
				break
			}
			if index == count {
				stop <- true
				errChan <- err
				break
			}
			index++
		}
	}(delayTime, retryFlag, stopRetry)

	var err error
	select {
	case <-stopRetry:
		for v := range retryFlag {
			err = v
			if v != nil {
				err = v
				break
			}
		}
	}
	close(stopRetry)
	if err != nil {
		close(retryFlag)
		return err
	}
	return nil
}

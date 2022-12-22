package beanq

import (
	"errors"
	"fmt"
	"log"
	"testing"
	"time"

	"beanq/helper/json"
	options2 "beanq/internal/options"

	"github.com/go-redis/redis/v8"
	"github.com/spf13/cast"
)

var (
	queue           = "ch2"
	group           = "g2"
	consumer        = "cs1"
	optionParameter options2.Options
)

func init() {
	optionParameter = options2.Options{
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
}

/*
  - TestPublishOne
  - @Description:
    publish one msg
  - @param t
*/
func TestPublishOne(t *testing.T) {

	msg := struct {
		Id   int
		Info string
	}{
		1,
		"msg------1",
	}

	d, _ := json.Marshal(msg)
	task := NewTask(SetPayLoad(d))

	err := Publish(task, options2.Queue("ch2"))
	if err != nil {
		t.Fatal(err.Error())
	}
	fmt.Printf("发布成功，消息：%+v \n", task)
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
		task := NewTask(SetPayLoad(d))

		err := Publish(task, options2.Queue("ch2"))

		if err != nil {
			log.Fatalln(err)
		}
		fmt.Printf("%+v \n", task)
	}

}

/*
  - TestDelayPublish
  - @Description:
    publish multiple schedule msgs
  - @param t
*/
func TestDelayPublish(t *testing.T) {
	pub := NewClient(NewRedisBroker(optionParameter.RedisOptions))

	m := make(map[string]string)

	for i := 0; i < 5; i++ {
		m["delayMsg"] = "new msg" + cast.ToString(i)
		b, _ := json.Marshal(m)

		task := NewTask(SetName("update"), SetPayLoad(b))
		delayT := time.Now().Add(10 * time.Second)

		res, err := pub.DelayPublish(task, delayT, options2.Queue("delay-ch"))
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Printf("%+v \n", res)
	}

	defer pub.Close()
}
func TestRetry(t *testing.T) {

	err := retry(func() error {
		fmt.Println("function body")
		return errors.New("错误")
		// return nil
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

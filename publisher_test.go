package beanq

import (
	"log"
	"testing"
	"time"

	"beanq/helper/json"
	opt "beanq/internal/options"

	"github.com/go-redis/redis/v8"
	"github.com/spf13/cast"
)

var (
	queue           = "ch2"
	group           = "g2"
	consumer        = "cs1"
	optionParameter opt.Options
)

func init() {
	optionParameter = opt.Options{
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
	task := NewTask(d)

	err := Publish(task, opt.Queue("ch2"), opt.Group("aa"))
	if err != nil {
		t.Fatal(err.Error())
	}
	Logger.Info(task)
}

/*
  - TestPublish
  - @Description:
    publisher
  - @param t
*/
func TestPublishMore(t *testing.T) {
	pub := NewClient()

	for i := 0; i < 5; i++ {
		m := make(map[int]string)
		m[i] = "k----" + cast.ToString(i)

		d, _ := json.Marshal(m)
		task := NewTask(d)

		res, err := pub.Publish(task, opt.Queue("delay-ch"))
		if err != nil {
			log.Fatalln(err)
		}
		Logger.Info(res)
	}
	t.Fatal(pub.Close())
}

/*
  - TestDelayPublish
  - @Description:
    publish multiple schedule msgs
  - @param t
*/
func TestDelayPublish(t *testing.T) {
	pub := NewClient()

	m := make(map[string]string)

	for i := 0; i < 5; i++ {
		y := 0
		m["delayMsg"] = "new msg" + cast.ToString(i)
		b, _ := json.Marshal(m)

		task := NewTask(b, SetName("update"))
		delayT := time.Now().Add(10 * time.Second)

		if i == 3 {
			y = 10
		}
		res, err := pub.DelayPublish(task, delayT, opt.Queue("delay-ch"), opt.Group("delay-group"), opt.Priority(float64(y)))
		if err != nil {
			log.Fatalln(err)
		}
		Logger.Info(res)
	}

	t.Fatal(pub.Close())
}

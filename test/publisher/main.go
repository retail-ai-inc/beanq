package main

import (
	"fmt"
	"log"
	"time"

	"beanq"
	"beanq/helper/json"
	options2 "beanq/internal/options"
	"github.com/go-redis/redis/v8"
	"github.com/spf13/cast"
)

func main() {
	pubOneInfo()
	// pubMoreInfo()
	// pubDelayInfo()
}
func pubOneInfo() {
	msg := struct {
		Id   int
		Info string
	}{
		1,
		"msg------1",
	}

	d, _ := json.Marshal(msg)
	task := beanq.NewTask(d)

	err := beanq.Publish(task, options2.Queue("ch2"), options2.Group("g2"))
	if err != nil {
		log.Fatal(err.Error())
	}
	fmt.Printf("SendMsgs：%+v \n", task)
}
func pubMoreInfo() {
	pub := beanq.NewClient(beanq.NewRedisBroker(&redis.Options{
		Addr:      beanq.Env.Queue.Redis.Host + ":" + cast.ToString(beanq.Env.Queue.Redis.Port),
		Dialer:    nil,
		OnConnect: nil,
		Username:  "",
		Password:  beanq.Env.Queue.Redis.Password,
		DB:        beanq.Env.Queue.Redis.Db,
	}))
	m := make(map[string]string)

	for i := 0; i < 5; i++ {
		y := 0
		m["delayMsg"] = "new msg" + cast.ToString(i)
		b, _ := json.Marshal(m)

		task := beanq.NewTask(b, beanq.SetName("update"))
		delayT := time.Now().Add(10 * time.Second)

		if i == 3 {
			y = 10
		}
		res, err := pub.DelayPublish(task, delayT, options2.Queue("delay-ch"), options2.Priority(float64(y)))
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Printf("%+v \n", res)
	}
	defer pub.Close()
}
func pubDelayInfo() {
	pub := beanq.NewClient(beanq.NewRedisBroker(&redis.Options{
		Addr:      beanq.Env.Queue.Redis.Host + ":" + cast.ToString(beanq.Env.Queue.Redis.Port),
		Dialer:    nil,
		OnConnect: nil,
		Username:  "",
		Password:  beanq.Env.Queue.Redis.Password,
		DB:        beanq.Env.Queue.Redis.Db,
	}))

	m := make(map[string]string)

	for i := 0; i < 5; i++ {
		y := 0
		m["delayMsg"] = "new msg" + cast.ToString(i)
		b, _ := json.Marshal(m)

		task := beanq.NewTask(b, beanq.SetName("update"))
		delayT := time.Now().Add(10 * time.Second)

		if i == 3 {
			y = 10
		}
		res, err := pub.DelayPublish(task, delayT, options2.Queue("delay-ch"), options2.Priority(float64(y)))
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Printf("%+v \n", res)
	}

	defer pub.Close()
}

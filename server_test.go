package beanq

import (
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/spf13/cast"
	"testing"
)

/*
  - TestPublishAndConsumer
  - @Description:
    Create 5~10 consumers in your local and publish just one job and see how the consumer took the job,
    one consumer took it or multiple consumers
  - @param t
*/
func TestPublishAndConsumer(t *testing.T) {
	rdb := NewBeanq("redis", options)
	//publish one job
	t.Run("publish", func(t *testing.T) {
		m := make(map[string]any)
		m["key1"] = "val1"
		b, _ := json.Marshal(m)
		task := NewTask("", b)
		result, err := rdb.Publish(task, Queue("pub"), Group("g1"))
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("PublishResult:%+v \n", result)
		rdb.Close()
	})
	//5~10 same consumers
	t.Run("10consumer", func(t *testing.T) {
		server := NewServer(10)
		for i := 0; i < 5; i++ {
			groupConsumer := "g1"
			server.Register(groupConsumer, "pub", func(task *Task, r *redis.Client) error {
				fmt.Printf(cast.ToString(i)+"-Payload:%s \n", task.Payload())
				return nil
			})
		}
		rdb.Start(server)
	})
	//5~10 different  consumers
	t.Run("10differentconsumer", func(t *testing.T) {
		server := NewServer(10)
		for i := 0; i < 5; i++ {
			groupConsumer := "g" + cast.ToString(i)
			server.Register(groupConsumer, "pub", func(task *Task, r *redis.Client) error {
				fmt.Printf(groupConsumer+"-Payload:%s \n", task.Payload())
				return nil
			})
		}
		rdb.Start(server)
	})
}

/*
  - TestConsumer
  - @Description:
    consumer
  - @param t
*/
func TestConsumer(t *testing.T) {
	rdb := NewBeanq("redis", options)

	server := NewServer(3)
	server.Register(group, queue, func(task *Task, r *redis.Client) error {

		fmt.Printf("PayLoad：%+v \n", task.Payload())
		return nil
	})
	server.Register(defaultOptions.defaultDelayGroup, defaultOptions.defaultDelayQueueName, func(task *Task, r *redis.Client) error {
		fmt.Printf("Delay:%+v \n", task.Payload())
		return nil
	})
	rdb.Start(server)

}
func TestConsumer2(t *testing.T) {

	rdb := NewBeanq("redis", options)

	server := NewServer(3)
	server.Register("g11", "c11", func(task *Task, r *redis.Client) error {
		fmt.Printf("2PayLoad:%+v \n", task.Payload())
		return nil
	})
	rdb.Start(server)
}
func TestDelayConsumer(t *testing.T) {
	rdb := NewRedis(options)
	rdb.delayConsumer()
}

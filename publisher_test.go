package beanq

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"beanq/helper/json"
	opt "beanq/internal/options"
	"github.com/spf13/cast"
)

func TestPublishOne(t *testing.T) {
	msg := struct {
		Id   int
		Info string
	}{
		1,
		"example: publish information",
	}

	d, _ := json.Marshal(msg)
	task := NewTask(d)

	pub := NewPublisher()
	err := pub.Publish(task, opt.Queue("ch"), opt.Group("group-one"))
	defer pub.Close()
	if err != nil {
		t.Fatal(err.Error())
	}
	Logger.Info(task)
}

func TestPublishMore(t *testing.T) {
	pub := NewPublisher()

	for i := 0; i < 5; i++ {
		m := make(map[int]string)
		m[i] = "publisher:" + cast.ToString(i)

		d, _ := json.Marshal(m)
		task := NewTask(d)

		err := pub.Publish(task, opt.Queue("delay-ch"))
		if err != nil {
			log.Fatalln(err)
		}
	}
	t.Fatal(pub.Close())
}

func TestDelayPublish(t *testing.T) {
	pub := NewPublisher()

	m := make(map[string]string)

	for i := 0; i < 50; i++ {
		y := 0
		m["delayMsg"] = "new msg" + cast.ToString(i)
		b, _ := json.Marshal(m)

		task := NewTask(b, SetName("update"))
		delayT := time.Now().Add(10 * time.Second)

		if i == 3 {
			y = 10
		}
		err := pub.DelayPublish(task, delayT, opt.Queue("delay-ch"), opt.Group("delay-group"), opt.Priority(float64(y)))
		if err != nil {
			log.Fatalln(err)
		}
	}

	defer pub.Close()
}
func TestSig(t *testing.T) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT, syscall.SIGTSTP)
	go func() {
		for {
			sig := <-sigs
			if sig.String() != "" {
				fmt.Println("aa")
			}
		}
	}()
	select {}
}

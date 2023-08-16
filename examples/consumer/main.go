package main

import (
	"net/http"
	_ "net/http/pprof"
	"time"

	"beanq"
)

func main() {

	go func() {
		http.ListenAndServe("0.0.0.0:8000", nil)
	}()

	// register consumer
	csm := beanq.NewConsumer()
	// register normal consumer
	csm.Register("g2", "ch2", func(task *beanq.Task) error {
		// TODO:logic
		// like this:
		time.Sleep(3 * time.Second) // this is my business.
		beanq.Logger.Info(task.Payload())

		return nil
	})
	// register delay consumer
	csm.Register("delay-group", "delay-ch", func(task *beanq.Task) error {
		beanq.Logger.Info(task.Payload())

		return nil
	})
	// begin to consume information
	csm.StartConsumer()
}

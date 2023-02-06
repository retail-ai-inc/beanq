package main

import (
	"os"
	"runtime/pprof"

	"beanq"
	"go.uber.org/zap"
)

func main() {

	f, err := os.Create("cpu.prof")
	if err != nil {
		beanq.Logger.Fatal("err", zap.Error(err))
	}
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()
	fm, err := os.Create("mem.prof")
	pprof.WriteHeapProfile(fm)

	// register consumer
	csm := beanq.NewConsumer()
	// register normal consumer
	csm.Register("g2", "ch2", func(task *beanq.Task) error {
		// TODO:logic
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

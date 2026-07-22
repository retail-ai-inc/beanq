package main

import (
	"context"
	"log"

	beanq "github.com/retail-ai-inc/beanq/v4"
)

func main() {
	config, err := beanq.NewConfig("./", "json", "env")
	if err != nil {
		log.Fatalf("Unable to create beanq config: %v", err)
	}
	csm := beanq.New(config)
	if err := csm.ServeHTTP(context.Background()); err != nil {
		log.Fatalf("Beanq UI stopped: %v", err)
	}
}

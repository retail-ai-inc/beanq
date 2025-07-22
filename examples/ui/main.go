package main

import (
	"context"
	"log"

	"github.com/retail-ai-inc/beanq/v4"
)

func main() {
	config, err := beanq.NewConfig("./", "json", "env")
	if err != nil {
		log.Fatalf("Unable to open beanq env.json file: %v", err)

	}
	csm := beanq.New(config)
	csm.ServeHttp(context.Background())
}

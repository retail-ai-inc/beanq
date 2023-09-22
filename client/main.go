package main

import (
	"log"

	"github.com/retail-ai-inc/beanq/client/internal/routers"
	"github.com/retail-ai-inc/beanq/client/internal/simple_router"
)

func main() {

	rt := simple_router.New()

	rt.Get("/", routers.IndexHandler)
	rt.Get("/schedule", routers.ScheduleHandler)
	rt.Get("/queue", routers.QueueHandler)
	rt.Get("/log", routers.Auth(routers.LogHandler))
	rt.Get("/redis", routers.RedisHandler)
	rt.Post("/login", routers.LoginHandler)
	rt.Delete("/log/del", routers.Auth(routers.LogDelHandler))
	rt.Post("/log/retry", routers.Auth(routers.LogRetryHandler))
	rt.Post("/log/archive", routers.Auth(routers.LogArchiveHandler))
	// rt.Get("/test", func(ctx *simple_router.Context) error {
	// 	fmt.Println("aa")
	// 	return nil
	// })
	// rt.Post("/test", func(ctx *simple_router.Context) error {
	// 	fmt.Println("bb")
	// 	return nil
	// })
	if err := rt.Run(":9090"); err != nil {
		log.Fatalln(err)
	}

}

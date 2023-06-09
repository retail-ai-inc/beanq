package main

import (
	"log"
	"net/http"

	"beanq/client/internal/routers"
)

func main() {

	mux := http.NewServeMux()

	mux.HandleFunc("/", routers.IndexHandler)
	mux.HandleFunc("/schedule", routers.ScheduleHandler)
	mux.HandleFunc("/queue", routers.QueueHandler)
	mux.HandleFunc("/redis", routers.RedisHandler)

	srv := &http.Server{
		Addr:    ":9090",
		Handler: mux,
	}
	log.Println("---start---")
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalln(err)
	}
}

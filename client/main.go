package main

import (
	"log"
	"net/http"

	"beanq/client/internal/routers"
)

func main() {

	srv := &http.Server{
		Addr:    ":9090",
		Handler: routers.ServeMux(),
	}
	log.Printf("----start----,listen on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalln(err)
	}
}

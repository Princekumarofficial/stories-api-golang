package main

import (
	"log"
	"net/http"

	"github.com/princekumarofficial/stories-service/internal/config"
)

func main() {
	// load config
	cfg := config.MustLoad()
	// database setup
	// setup server
	router := http.NewServeMux()

	router.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, World!"))
	})
	// setup router

	server := http.Server{
		Addr:    cfg.HTTPServer.Address,
		Handler: router,
	}

	log.Println("server started on", cfg.HTTPServer.Address)
	err := server.ListenAndServe()
	if err != nil {
		log.Fatalf("failed to start server: %s", err)
	}
}

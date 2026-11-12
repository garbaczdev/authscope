package main

import (
	"log"
	"net/http"

	"github.com/garbaczdev/authscope/admin"
	"github.com/garbaczdev/authscope/store"
)


func main() {
	reg, err := store.NewRegistry("./data/store.db")
	if err != nil {
		log.Fatalf("could not create registry: %v", err)
	}

	mux, err := admin.New(reg)
	if err != nil {
		log.Fatalf("could not create admin multiplexer: %v", err)
	}

	server := &http.Server{
		Addr: ":8080",
		Handler: mux,
	}

	log.Println("Server running on http://localhost:8080")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}

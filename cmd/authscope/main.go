package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"flag"

	"github.com/garbaczdev/authscope/admin"
	"github.com/garbaczdev/authscope/store"
	"github.com/garbaczdev/authscope/verify"
)

func envStr(key, def string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

func main() {
	var (
		adminAddr = flag.String("admin-addr", envStr("ADMIN_ADDR", "0.0.0.0:8080"), "address to expose the admin panel to")
		verifyAddr = flag.String("verify-addr", envStr("VERIFY_ADDR", "0.0.0.0:8081"), "address to expose the verify interface to")
	)
	flag.Parse()

	reg, err := store.NewRegistry("./data/store.db")
	if err != nil {
		log.Fatalf("could not create registry: %v", err)
	}

	adminHandler, err := admin.New(reg)
	if err != nil {
		log.Fatalf("could not create admin multiplexer: %v", err)
	}

	verifyHandler, err := verify.New(reg)
	if err != nil {
		log.Fatalf("could not create verify multiplexer: %v", err)
	}

	// The verify endpoints are the internal forward-auth surface served on
	// their own listener; the admin UI runs on a separate port.
	adminServer := &http.Server{
		Addr:    *adminAddr,
		Handler: adminHandler,
	}
	verifyServer := &http.Server{
		Addr:    *verifyAddr,
		Handler: verifyHandler,
	}

	serverErr := make(chan error, 2)
	go func() {
		log.Printf("admin server listening on http://%s", *adminAddr)
		if err := adminServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- fmt.Errorf("admin server: %w", err)
		}
	}()
	go func() {
		log.Printf("verify server listening on http://%s", *verifyAddr)
		if err := verifyServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- fmt.Errorf("verify server: %w", err)
		}
	}()

	log.Fatalf("%v", <-serverErr)
}

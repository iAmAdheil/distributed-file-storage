package main

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"
)

func (fServer *FileServer) startHttpServer() {
	mux := http.NewServeMux()

	// Register handlers using HandleFunc (for simple functions)
	mux.HandleFunc("GET /health", fServer.healthHandler)
	mux.HandleFunc("PUT /{key}", fServer.storeHandler)
	mux.HandleFunc("GET /{key}", fServer.getHandler)

	srv := &http.Server{
		Addr:    fServer.HttpAddr,
		Handler: mux, // <-- This is where you hand the receptionist (mux) over to the building (srv)
	}

	// Pass the mux to http.ListenAndServe
	fmt.Printf("Server starting on %s...\n", fServer.HttpAddr)
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			fmt.Printf("Server (%s) start with err: %v\n", fServer.HttpAddr, err)
		}
	}()

	shutdown, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	<-shutdown.Done()

	fmt.Printf("Shutting down server %s...\n", fServer.HttpAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		fmt.Printf("Server (%s) shutdown with err: %v\n", fServer.HttpAddr, err)
	}
	fmt.Printf("Shutdown complete (%s).\n", fServer.HttpAddr)
}

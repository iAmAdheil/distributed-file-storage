package main

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"
)

func validateURLParamsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bucket := r.PathValue("bucket")
		key := r.PathValue("key")

		if len(key) == 0 || len(bucket) == 0 {
			http.Error(w, "bucket and key are compulsary URL param fields.", http.StatusBadRequest)
			return
		}

		// API key is valid, proceed to next handler
		next.ServeHTTP(w, r)
	})
}

func (fServer *FileServer) startHttpServer() {
	mux := http.NewServeMux()

	// Register handlers using HandleFunc (for simple functions)
	mux.HandleFunc("GET /health", fServer.healthHandler)
	mux.HandleFunc("PUT /{bucket}/{key}", validateURLParamsMiddleware(fServer.storeHandler))
	mux.HandleFunc("GET /{bucket}/{key}", validateURLParamsMiddleware(fServer.getHandler))
	mux.HandleFunc("DELETE /{bucket}/{key}", validateURLParamsMiddleware(fServer.deleteHandler))

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
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		fmt.Printf("Server (%s) shutdown with err: %v\n", fServer.HttpAddr, err)
	}
	fmt.Printf("Shutdown complete (%s).\n", fServer.HttpAddr)
}

package main

import (
	"fmt"
	"net/http"
)

type HttpOpts struct {
	HttpAddr string
}

type HttpServer struct {
	HttpOpts

	Quitch chan struct{}
}

func NewHttpServer(opts HttpOpts) *HttpServer {
	return &HttpServer{
		HttpOpts: opts,
		Quitch:   make(chan struct{}),
	}
}

func (hs *HttpServer) Start() {
	mux := http.NewServeMux()

	// Register handlers using HandleFunc (for simple functions)
	mux.HandleFunc("GET /health", healthHandler)

	srv := &http.Server{
		Addr:    hs.HttpAddr,
		Handler: mux, // <-- This is where you hand the receptionist (mux) over to the building (srv)
	}

	// 3. Pass the mux to http.ListenAndServe
	fmt.Printf("Server starting on %s...\n", hs.HttpAddr)
	err := srv.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	fmt.Fprint(w, "Server ready to respond!\n")
}

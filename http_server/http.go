package http_server

// package http involves asset maintenance
// all imp methods -> storage, replication is handled by the internal server
// http is just an entrypoint

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/iAmAdheil/distributed-file-storage/db"
)

type InternalFileServer interface {
	Get(key string) (io.ReadCloser, error)
	Store(key string, r io.Reader) error
	Delete(key string) error
}

type HTTPServerOpts struct {
	ListenAddress string
	Internal      InternalFileServer
}

type HTTPServer struct {
	HTTPServerOpts

	httpAddr string
	db       *db.DB
}

func New(httpOpts HTTPServerOpts) *HTTPServer {
	dbOpts := db.DBOpts{
		Filename: "http_" + httpOpts.ListenAddress + ".db",
	}
	db := db.New(dbOpts)

	return &HTTPServer{
		HTTPServerOpts: httpOpts,

		db:       db,
		httpAddr: httpOpts.ListenAddress,
	}
}

func (server *HTTPServer) Address() string {
	return server.httpAddr
}

func (server *HTTPServer) Start() error {
	go server.Listen()
	return nil
}

func (server *HTTPServer) Listen() {
	mux := http.NewServeMux()

	// Register handlers using HandleFunc (for simple functions)
	mux.HandleFunc("GET /health", setResHeaders(server.healthHandler))
	mux.HandleFunc("PUT /{bucket}/{key}", setResHeaders(validateURLParamsMiddleware(server.storeHandler, []string{"bucket", "key"})))
	// get handles setting its own content headers -> streams raw bytes, no XML returned
	mux.HandleFunc("GET /{bucket}/{key}", validateURLParamsMiddleware(server.getHandler, []string{"bucket", "key"}))
	mux.HandleFunc("DELETE /{bucket}/{key}", setResHeaders(validateURLParamsMiddleware(server.deleteHandler, []string{"bucket", "key"})))
	mux.HandleFunc("GET /{bucket}", setResHeaders(validateURLParamsMiddleware(server.listHandler, []string{"bucket"})))

	srv := &http.Server{
		Addr:    server.httpAddr,
		Handler: mux, // <-- This is where you hand the receptionist (mux) over to the building (srv)
	}

	// Pass the mux to http.ListenAndServe
	fmt.Printf("Server starting on %s...\n", server.httpAddr)
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			fmt.Printf("Server (%s) start with err: %v\n", server.httpAddr, err)
		}
	}()

	shutdown, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	<-shutdown.Done()

	fmt.Printf("Shutting down server %s...\n", server.httpAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		fmt.Printf("Server (%s) shutdown with err: %v\n", server.httpAddr, err)
	}
	fmt.Printf("Shutdown complete (%s).\n", server.httpAddr)
}

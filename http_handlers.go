package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
)

func (fServer *FileServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	fmt.Fprint(w, "Server ready to respond!\n")
}

func (fServer *FileServer) storeHandler(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if len(key) == 0 {
		http.Error(w, "Key is required.", http.StatusBadRequest)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, MAX_FILE_SIZE)

	if err := fServer.Store(key, r.Body); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			http.Error(w, "Request body is too large (Max 1MB allowed)", http.StatusRequestEntityTooLarge)
			return
		}

		http.Error(w, "An error occured when storing the file.", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("File stored successfully."))
}

func (fServer *FileServer) getHandler(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if len(key) == 0 {
		http.Error(w, "Key is required.", http.StatusBadRequest)
		return
	}

	cr, err := fServer.Get(key) // content reader
	if err != nil {
		http.Error(w, "File not found.", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusOK)

	bw, err := io.Copy(w, cr)
	if err != nil {
		// PHASE 2 ERROR: Streaming already started.
		// w.WriteHeader() has already fired. We can only log this.
		log.Printf("Error mid-stream after sending %d bytes: %v", bw, err)
		return
	}
}

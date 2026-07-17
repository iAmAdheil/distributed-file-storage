package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/iAmAdheil/distributed-file-storage/db/model"
)

func (fServer *FileServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	fmt.Fprint(w, "Server ready to respond!\n")
}

func (fServer *FileServer) storeHandler(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	bucket := r.PathValue("bucket")
	pathkey := bucket + "/" + key

	r.Body = http.MaxBytesReader(w, r.Body, MAX_FILE_SIZE)

	fileMetadata := &model.Metadata{
		Bucket:      bucket,
		Key:         key,
		Size:        r.ContentLength,
		ContentType: r.Header.Get("Content-Type"),
		CreatedAt:   time.Now(),
	}

	if err := fServer.Store(pathkey, r.Body); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			http.Error(w, "Request body is too large (Max 1MB allowed)", http.StatusRequestEntityTooLarge)
			return
		}

		http.Error(w, "An error occured when storing the file.", http.StatusInternalServerError)
		return
	}

	if err := fServer.db.PutMeta(fileMetadata); err != nil {
		http.Error(w, "An error occured when storing the file.", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("File stored successfully."))
}

func (fServer *FileServer) getHandler(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	bucket := r.PathValue("bucket")
	pathkey := bucket + "/" + key

	cr, err := fServer.Get(pathkey) // content reader
	if err != nil {
		http.Error(w, "File not found.", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Transfer-Encoding", "chunked")

	w.WriteHeader(http.StatusOK)

	bw, err := io.Copy(w, cr)
	if err != nil {
		// PHASE 2 ERROR: Streaming already started.
		// w.WriteHeader() has already fired. We can only log this.
		log.Printf("Error mid-stream after sending %d bytes: %v", bw, err)
		return
	}
}

func (fServer *FileServer) deleteHandler(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	bucket := r.PathValue("bucket")
	pathkey := bucket + "/" + key

	if err := fServer.Delete(pathkey); err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
	}

	fServer.db.DeleteMeta(bucket, key)

	w.WriteHeader(http.StatusNoContent)
}

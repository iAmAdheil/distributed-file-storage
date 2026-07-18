package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/iAmAdheil/distributed-file-storage/db"
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
		return
	}

	fServer.db.DeleteMeta(bucket, key)

	w.WriteHeader(http.StatusNoContent)
}

type listhandlerRes struct {
	Objects     []model.Metadata `json:"objects"`
	IsTruncated bool             `json:"is_truncated"`
	ContToken   string           `json:"continuation_token"`
}

func (fServer *FileServer) listHandler(w http.ResponseWriter, r *http.Request) {
	bucket := r.PathValue("bucket")
	prefix := r.URL.Query().Get("prefix")
	maxKeys, err := strconv.Atoi(r.URL.Query().Get("max-keys"))
	if err != nil {
		maxKeys = -1
	}
	contToken := r.URL.Query().Get("continuation-token")

	params := db.ListMetaParams{
		Prefix:    prefix,
		MaxKeys:   maxKeys,
		ContToken: contToken,
	}

	res, err := fServer.db.ListMeta(bucket, params)
	if err != nil {
		http.Error(w, "Failed to fetch bucket items.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	isTrunc := false
	if len(res.ContToken) > 0 {
		isTrunc = true
	}

	if err := json.NewEncoder(w).Encode(listhandlerRes{Objects: res.List, IsTruncated: isTrunc, ContToken: res.ContToken}); err != nil {
		fmt.Printf("Error when encoding list items: %s\n", err.Error())
	}
}

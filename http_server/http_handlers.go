package http_server

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/iAmAdheil/distributed-file-storage/db"
	"github.com/iAmAdheil/distributed-file-storage/db/model"
)

// bucket and key checks should be shifted to a validation layer before more important busi. logic
// (@iAmAdheil), will help debloat controllers
// controllers -> only busi. logic, no validation should be here
// should be handled from inside of middlewares

func (server *HTTPServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
}

func (server *HTTPServer) storeHandler(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	bucket := r.PathValue("bucket")
	pathkey := bucket + "/" + key

	r.Body = http.MaxBytesReader(w, r.Body, MAX_FILE_SIZE)

	w.Header().Set("Content-Type", "application/xml")

	fileMetadata := &model.Metadata{
		Bucket:      bucket,
		Key:         key,
		Size:        r.ContentLength,
		ContentType: r.Header.Get("Content-Type"),
		CreatedAt:   time.Now(),
	}

	// stores the content hash for etag
	chash := md5.New()

	s3errOpts := &s3ErrOpts{
		Bucket: bucket,
		Key:    key,
	}

	if err := server.Internal.Store(pathkey, io.TeeReader(r.Body, chash)); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			WriteS3Err(w, http.StatusRequestEntityTooLarge, "EntityTooLarge", s3errOpts)
			return
		}

		s3errOpts.Msg = "An error occured while storing the file"
		WriteS3Err(w, http.StatusInternalServerError, "InternalError", s3errOpts)
		return
	}

	if err := server.db.PutMeta(fileMetadata); err != nil {
		s3errOpts.Msg = "An error occured while storing file metadata"
		WriteS3Err(w, http.StatusInternalServerError, "InternalError", s3errOpts)
		return
	}

	etag := chash.Sum(nil)

	w.Header().Set("ETag", hex.EncodeToString(etag[:]))
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("File stored successfully."))
}

func (server *HTTPServer) getHandler(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	bucket := r.PathValue("bucket")
	pathkey := bucket + "/" + key

	w.Header().Set("Content-Type", "application/xml")

	s3errOpts := &s3ErrOpts{
		Bucket: bucket,
		Key:    key,
	}

	md, err := server.db.GetMeta(bucket, key)
	if err != nil {
		var bnf *db.BucketNotFound
		if errors.As(err, &bnf) {
			s3errOpts.Msg = err.Error()
			WriteS3Err(w, http.StatusNotFound, "NoSuchBucket", s3errOpts)
			return
		}

		var knf *db.KeyNotFound
		if errors.As(err, &knf) {
			s3errOpts.Msg = err.Error()
			WriteS3Err(w, http.StatusNotFound, "NoSuchKey", s3errOpts)
			return
		}

		s3errOpts.Msg = "File metadata could not be retrieved"
		WriteS3Err(w, http.StatusInternalServerError, "InternalError", s3errOpts)
		return
	}

	cr, err := server.Internal.Get(pathkey) // content reader
	if err != nil {
		// ignoring any replication errors, only handling if file not found locally
		// before and after replication
		var pathErr *os.PathError
		if errors.As(err, &pathErr) {
			s3errOpts.Msg = "File not found"
			WriteS3Err(w, http.StatusNotFound, "InternalError", s3errOpts)
			return
		}

		s3errOpts.Msg = "File could not be retrieved"
		WriteS3Err(w, http.StatusInternalServerError, "InternalError", s3errOpts)
		return
	}
	defer cr.Close()

	br := bufio.NewReader(cr)

	var contentType string = md.ContentType
	if len(contentType) == 0 {
		m := mime.TypeByExtension(key)
		if len(m) > 0 {
			contentType = m
		} else {
			pb, err := br.Peek(512) // peek bytes
			if err != nil && err != io.EOF {
				s3errOpts.Msg = "File mimetype/extension could not be deciphered"
				WriteS3Err(w, http.StatusInternalServerError, "InternalError", s3errOpts)
				return
			}

			contentType = http.DetectContentType(pb)
		}
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.FormatInt(md.Size, 10))
	w.Header().Set("Last-Modified", md.CreatedAt.UTC().Format(http.TimeFormat))

	w.WriteHeader(http.StatusOK)

	bw, err := io.Copy(w, br)
	if err != nil {
		// PHASE 2 ERROR: Streaming already started.
		// w.WriteHeader() has already fired. We can only log this.
		log.Printf("Error mid-stream after sending %d bytes: %v", bw, err)
		return
	}
}

func (server *HTTPServer) deleteHandler(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	bucket := r.PathValue("bucket")
	pathkey := bucket + "/" + key

	s3errOpts := &s3ErrOpts{
		Bucket: bucket,
		Key:    key,
	}

	_, err := server.db.GetMeta(bucket, key)
	if err != nil {
		var bnf *db.BucketNotFound
		if errors.As(err, &bnf) {
			s3errOpts.Msg = err.Error()
			WriteS3Err(w, http.StatusNotFound, "NoSuchBucket", s3errOpts)
			return
		}

		var knf *db.KeyNotFound
		if errors.As(err, &knf) {
			s3errOpts.Msg = err.Error()
			WriteS3Err(w, http.StatusNotFound, "NoSuchKey", s3errOpts)
			return
		}

		s3errOpts.Msg = "File metadata could not be retrieved"
		WriteS3Err(w, http.StatusInternalServerError, "InternalError", s3errOpts)
		return
	}

	if err := server.Internal.Delete(pathkey); err != nil {
		s3errOpts.Msg = "File not found"
		WriteS3Err(w, http.StatusNotFound, "InternalError", s3errOpts)
		return
	}

	if err := server.db.DeleteMeta(bucket, key); err != nil {
		fmt.Printf("Delete metadata failed: %s\n", err.Error())
	}

	w.WriteHeader(http.StatusNoContent)
}

type listhandlerRes struct {
	Objects     []model.Metadata `json:"objects"`
	IsTruncated bool             `json:"is_truncated"`
	ContToken   string           `json:"continuation_token"`
}

func (server *HTTPServer) listHandler(w http.ResponseWriter, r *http.Request) {
	bucket := r.PathValue("bucket")
	prefix := r.URL.Query().Get("prefix")
	maxKeys, err := strconv.Atoi(r.URL.Query().Get("max-keys"))
	if err != nil {
		maxKeys = -1
	}
	contToken := r.URL.Query().Get("continuation-token")

	w.Header().Set("Content-Type", "application/json")

	params := db.ListMetaParams{
		Prefix:    prefix,
		MaxKeys:   maxKeys,
		ContToken: contToken,
	}

	s3errOpts := &s3ErrOpts{
		Bucket: bucket,
	}

	res, err := server.db.ListMeta(bucket, params)
	if err != nil {
		var bnf *db.BucketNotFound
		if errors.As(err, &bnf) {
			s3errOpts.Msg = err.Error()
			WriteS3Err(w, http.StatusNotFound, "NoSuchBucket", s3errOpts)
			return
		}

		s3errOpts.Msg = "Failed to fetch bucket items"
		WriteS3Err(w, http.StatusInternalServerError, "InternalError", s3errOpts)
		return
	}

	w.WriteHeader(http.StatusOK)

	isTrunc := false
	if len(res.ContToken) > 0 {
		isTrunc = true
	}

	if err := json.NewEncoder(w).Encode(listhandlerRes{Objects: res.List, IsTruncated: isTrunc, ContToken: res.ContToken}); err != nil {
		fmt.Printf("Error when encoding list items: %s\n", err.Error())
	}
}

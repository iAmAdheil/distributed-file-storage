package main

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
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

type s3error struct {
	XMLName xml.Name `xml:"Error"`

	Code     string `xml:"Code"`
	Message  string `xml:"Message"`
	Bucket   string `xml:"Bucket,omitempty"`
	Key      string `xml:"Key,omitempty"`
	Resource string `xml:"Resource,omitempty"`
	ReqId    string `xml:"RequestId,omitempty"`
}

func (s *s3error) WriteS3Err(w http.ResponseWriter, status int) {
	w.WriteHeader(status)
	xml.NewEncoder(w).Encode(*s)
}

func (fServer *FileServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	fmt.Fprint(w, "Server ready to respond!\n")
}

func (fServer *FileServer) storeHandler(w http.ResponseWriter, r *http.Request) {
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

	if err := fServer.Store(pathkey, io.TeeReader(r.Body, chash)); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			s3err := &s3error{
				Code:     "EntityTooLarge",
				Message:  "Request body is too large (Max 1MB allowed)",
				Bucket:   bucket,
				Key:      key,
				Resource: fmt.Sprintf("/%s/%s", bucket, key),
			}
			s3err.WriteS3Err(w, http.StatusRequestEntityTooLarge)
			return
		}

		s3err := &s3error{
			Code:     "InternalError",
			Message:  "An error occured while storing the file",
			Bucket:   bucket,
			Key:      key,
			Resource: fmt.Sprintf("/%s/%s", bucket, key),
		}
		s3err.WriteS3Err(w, http.StatusInternalServerError)
		return
	}

	if err := fServer.db.PutMeta(fileMetadata); err != nil {
		s3err := &s3error{
			Code:     "InternalError",
			Message:  "An error occured while storing file metadata",
			Bucket:   bucket,
			Key:      key,
			Resource: fmt.Sprintf("/%s/%s", bucket, key),
		}
		s3err.WriteS3Err(w, http.StatusInternalServerError)
		return
	}

	etag := chash.Sum(nil)

	w.Header().Set("ETag", hex.EncodeToString(etag[:]))
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("File stored successfully."))
}

func (fServer *FileServer) getHandler(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	bucket := r.PathValue("bucket")
	pathkey := bucket + "/" + key

	w.Header().Set("Content-Type", "application/xml")

	md, err := fServer.db.GetMeta(bucket, key)
	if err != nil {
		var bnf *db.BucketNotFound
		if errors.As(err, &bnf) {
			s3err := &s3error{
				Code:     "NoSuchBucket",
				Message:  err.Error(),
				Bucket:   bucket,
				Key:      key,
				Resource: fmt.Sprintf("/%s/%s", bucket, key),
			}
			s3err.WriteS3Err(w, http.StatusNotFound)
			return
		}

		var knf *db.KeyNotFound
		if errors.As(err, &knf) {
			s3err := &s3error{
				Code:     "NoSuchKey",
				Message:  err.Error(),
				Bucket:   bucket,
				Key:      key,
				Resource: fmt.Sprintf("/%s/%s", bucket, key),
			}
			s3err.WriteS3Err(w, http.StatusNotFound)
			return
		}

		s3err := &s3error{
			Code:     "InternalError",
			Message:  "File metadata could not be retrieved",
			Bucket:   bucket,
			Key:      key,
			Resource: fmt.Sprintf("/%s/%s", bucket, key),
		}
		s3err.WriteS3Err(w, http.StatusInternalServerError)
		return
	}

	cr, err := fServer.Get(pathkey) // content reader
	if err != nil {
		// ignoring any replication errors, only handling if file not found locally
		// before and after replication
		var pathErr *os.PathError
		if errors.As(err, &pathErr) {
			s3err := &s3error{
				Code:     "InternalError",
				Message:  "File not found",
				Bucket:   bucket,
				Key:      key,
				Resource: fmt.Sprintf("/%s/%s", bucket, key),
			}
			s3err.WriteS3Err(w, http.StatusNotFound)
			return
		}

		s3err := &s3error{
			Code:     "InternalError",
			Message:  "File could not be retrieved",
			Bucket:   bucket,
			Key:      key,
			Resource: fmt.Sprintf("/%s/%s", bucket, key),
		}
		s3err.WriteS3Err(w, http.StatusInternalServerError)
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
				s3err := &s3error{
					Code:     "InternalError",
					Message:  "File mimetype/extension could not be deciphered",
					Bucket:   bucket,
					Key:      key,
					Resource: fmt.Sprintf("/%s/%s", bucket, key),
				}
				s3err.WriteS3Err(w, http.StatusInternalServerError)
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

func (fServer *FileServer) deleteHandler(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	bucket := r.PathValue("bucket")
	pathkey := bucket + "/" + key

	// confirm if bucket and key exist
	_, err := fServer.db.GetMeta(bucket, key)
	if err != nil {
		var bnf *db.BucketNotFound
		if errors.As(err, &bnf) {
			s3err := &s3error{
				Code:     "NoSuchBucket",
				Message:  err.Error(),
				Bucket:   bucket,
				Key:      key,
				Resource: fmt.Sprintf("/%s/%s", bucket, key),
			}
			s3err.WriteS3Err(w, http.StatusNotFound)
			return
		}

		var knf *db.KeyNotFound
		if errors.As(err, &knf) {
			s3err := &s3error{
				Code:     "NoSuchKey",
				Message:  err.Error(),
				Bucket:   bucket,
				Key:      key,
				Resource: fmt.Sprintf("/%s/%s", bucket, key),
			}
			s3err.WriteS3Err(w, http.StatusNotFound)
			return
		}

		s3err := &s3error{
			Code:     "InternalError",
			Message:  "File metadata could not be retrieved",
			Bucket:   bucket,
			Key:      key,
			Resource: fmt.Sprintf("/%s/%s", bucket, key),
		}
		s3err.WriteS3Err(w, http.StatusInternalServerError)
		return
	}

	if err := fServer.Delete(pathkey); err != nil {
		s3err := &s3error{
			Code:     "InternalError",
			Message:  "File not found",
			Bucket:   bucket,
			Key:      key,
			Resource: fmt.Sprintf("/%s/%s", bucket, key),
		}
		s3err.WriteS3Err(w, http.StatusNotFound)
		return
	}

	if err := fServer.db.DeleteMeta(bucket, key); err != nil {
		// case -> file deleted but metadata deletion failed
		// (@iAmAdheil) handle tthis in the future
		fmt.Printf("Delete metadata failed: %s\n", err.Error())
	}

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

	w.Header().Set("Content-Type", "application/json")

	params := db.ListMetaParams{
		Prefix:    prefix,
		MaxKeys:   maxKeys,
		ContToken: contToken,
	}

	res, err := fServer.db.ListMeta(bucket, params)
	if err != nil {
		var bnf *db.BucketNotFound
		if errors.As(err, &bnf) {
			if errors.As(err, &bnf) {
				s3err := &s3error{
					Code:     "NoSuchBucket",
					Message:  err.Error(),
					Bucket:   bucket,
					Resource: fmt.Sprintf("/%s", bucket),
				}
				s3err.WriteS3Err(w, http.StatusNotFound)
				return
			}
			return
		}

		s3err := &s3error{
			Code:     "InternalError",
			Message:  "Failed to fetch bucket items",
			Bucket:   bucket,
			Resource: fmt.Sprintf("/%s", bucket),
		}
		s3err.WriteS3Err(w, http.StatusInternalServerError)
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

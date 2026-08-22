package http_server

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
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

	// compute etag on content hash, encode to string
	etag := hex.EncodeToString(chash.Sum(nil)[:])
	fileMetadata.ETag = etag

	if err := server.db.PutMeta(fileMetadata); err != nil {
		// best effort in case put meta fails
		server.Internal.Delete(pathkey)

		s3errOpts.Msg = "An error occured while storing file metadata"
		WriteS3Err(w, http.StatusInternalServerError, "InternalError", s3errOpts)
		return
	}

	w.Header().Set("ETag", etag)
	w.WriteHeader(http.StatusOK)
}

func (server *HTTPServer) getHandler(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	bucket := r.PathValue("bucket")
	pathkey := bucket + "/" + key

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

	if noneMatch := r.Header.Get("If-None-Match"); noneMatch == md.ETag {
		w.Header().Set("ETag", md.ETag)
		w.WriteHeader(http.StatusNotModified)
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

	w.Header().Set("ETag", md.ETag)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.FormatInt(md.Size, 10))
	w.Header().Set("Last-Modified", md.CreatedAt.UTC().Format(http.TimeFormat))

	w.WriteHeader(http.StatusOK)

	bw, err := io.Copy(w, br)
	if err != nil {
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
	XMLName xml.Name `xml:"ListBucketResult"`

	Name      string `xml:"Name"`
	Prefix    string `xml:"Prefix,omitempty"`
	KeyCount  string `xml:"KeyCount"`
	MaxKeys   string `xml:"MaxKeys"`
	IsTrunc   bool   `xml:"IsTruncated"`
	ContToken string `xml:"ContinuationToken,omitempty"`

	Objects []model.Metadata `xml:"Contents"`
}

func (server *HTTPServer) listHandler(w http.ResponseWriter, r *http.Request) {
	bucket := r.PathValue("bucket")
	prefix := r.URL.Query().Get("prefix")
	maxKeys, err := strconv.Atoi(r.URL.Query().Get("max-keys"))
	if err != nil {
		// set default max keys
		maxKeys = 1000
	}
	contToken := r.URL.Query().Get("continuation-token")

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

	if err := xml.NewEncoder(w).Encode(listhandlerRes{
		Name:      bucket,
		Prefix:    prefix,
		KeyCount:  strconv.FormatInt(int64(len(res.List)), 10),
		MaxKeys:   strconv.FormatInt(int64(maxKeys), 10),
		IsTrunc:   isTrunc,
		ContToken: res.ContToken,

		Objects: res.List,
	}); err != nil {
		fmt.Printf("Error when encoding list items: %s\n", err.Error())
	}
}

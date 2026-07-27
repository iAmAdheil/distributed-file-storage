package http_server

import (
	"encoding/xml"
	"fmt"
	"net/http"
)

type s3ErrOpts struct {
	Bucket string
	Key    string
	Msg    string
}

type s3error struct {
	XMLName xml.Name `xml:"Error"`

	Code     string `xml:"Code"`
	Message  string `xml:"Message"`
	Bucket   string `xml:"Bucket,omitempty"`
	Key      string `xml:"Key,omitempty"`
	Resource string `xml:"Resource,omitempty"`
	ReqId    string `xml:"RequestId,omitempty"`
}

func getS3Error(code string, s3Opts *s3ErrOpts) s3error {
	s3err := s3error{
		Bucket:   s3Opts.Bucket,
		Key:      s3Opts.Key,
		Resource: fmt.Sprintf("/%s/%s", s3Opts.Bucket, s3Opts.Key),
	}

	switch code {
	case "EntityTooLarge":
		s3err.Message = "Request body is too large (Max 1MB allowed)"
	case "NoSuchBucket":
		s3err.Message = s3Opts.Msg
	case "NoSuchKey":
		s3err.Message = s3Opts.Msg
	default:
		s3err.Code = "InternalError"
		s3err.Message = s3Opts.Msg
	}

	return s3err
}

func WriteS3Err(w http.ResponseWriter, status int, code string, s3ErrOpts *s3ErrOpts) {
	w.WriteHeader(status)
	err := getS3Error(code, s3ErrOpts)
	xml.NewEncoder(w).Encode(err)
}

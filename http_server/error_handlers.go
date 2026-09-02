package http_server

import (
	"encoding/xml"
	"net/http"
)

type s3ErrOpts struct {
	Code     string
	Bucket   string
	Key      string
	Msg      string
	Resource string
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

func getS3Error(s3Opts *s3ErrOpts) s3error {
	s3err := s3error{
		Bucket:   s3Opts.Bucket,
		Key:      s3Opts.Key,
		Code:     s3Opts.Code,
		Resource: s3Opts.Resource,
	}

	switch {
	case len(s3Opts.Msg) > 0:
		s3err.Message = s3Opts.Msg

	case s3Opts.Code == "EntityTooLarge":
		s3err.Message = "Request body is too large (Max 1MB allowed)"
	case s3Opts.Code == "NoSuchBucket":
		s3err.Message = "Bucket does not exist"
	case s3Opts.Code == "NoSuchKey":
		s3err.Message = "File with key does not exist"
	case s3Opts.Code == "InvalidBucketName":
		s3err.Message = "Bucket name is invalid"

	default:
		s3err.Code = "InternalError"
		s3err.Message = "Some internal error occured"
	}

	return s3err
}

func WriteS3Err(w http.ResponseWriter, status int, s3ErrOpts *s3ErrOpts) {
	// for routes that don't return xml normally, only for errors
	w.Header().Set("Content-Type", "application/xml")

	w.WriteHeader(status)
	err := getS3Error(s3ErrOpts)
	xml.NewEncoder(w).Encode(err)
}

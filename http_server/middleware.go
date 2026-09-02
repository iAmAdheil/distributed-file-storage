package http_server

import (
	"net/http"
	"strings"
)

func validateURLParamsMiddleware(next http.HandlerFunc, params []string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, param := range params {
			val := r.PathValue(param)
			var s3errOpts *s3ErrOpts

			switch param {
			case "key":
				s3errOpts = validateKey(val)
				if s3errOpts != nil {
					s3errOpts.Key = val
				}
			case "bucket":
				s3errOpts = validateBucket(val)
				if s3errOpts != nil {
					s3errOpts.Bucket = val
				}
			}

			if s3errOpts != nil {
				WriteS3Err(w, http.StatusBadRequest, s3errOpts)
				return
			}
		}

		// API key is valid, proceed to next handler
		next.ServeHTTP(w, r)
	})
}

func validateKey(key string) *s3ErrOpts {
	var s3errOpts *s3ErrOpts = nil

	if len(key) == 0 {
		s3errOpts = &s3ErrOpts{
			Code: "NoSuchKey",
		}
	}

	return s3errOpts
}

func validateBucket(bucket string) *s3ErrOpts {
	var s3errOpts *s3ErrOpts = nil

	if len(bucket) == 0 {
		s3errOpts = &s3ErrOpts{
			Code: "NoSuchBucket",
		}
	} else if strings.Contains(bucket, "/") {
		s3errOpts = &s3ErrOpts{
			Code: "InvalidBucketName",
		}
	}

	return s3errOpts
}

func setResHeaders(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")

		// API key is valid, proceed to next handler
		next.ServeHTTP(w, r)
	})
}

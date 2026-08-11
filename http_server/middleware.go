package http_server

import (
	"fmt"
	"net/http"
)

func validateURLParamsMiddleware(next http.HandlerFunc, params []string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, param := range params {
			val := r.PathValue(param)
			if len(val) == 0 {
				errMsg := fmt.Sprintf("%s is a compulsary URL param field.", param)
				http.Error(w, errMsg, http.StatusBadRequest)
				return
			}
		}

		// API key is valid, proceed to next handler
		next.ServeHTTP(w, r)
	})
}

func setResHeaders(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/xml")

		// API key is valid, proceed to next handler
		next.ServeHTTP(w, r)
	})
}

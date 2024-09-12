package router

import (
	"encoding/json"
	"net/http"
	"simple-mock-server/internal/mock"
)

func RegisterMocks(mocks []mock.Mock) {
	for _, mock := range mocks {
		for _, path := range mock.Paths {
			http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == mock.Verb {
					response := mock.Body
					for hk, hv := range mock.Headers {
						w.Header().Set(hk, hv)
					}
					w.WriteHeader(mock.Status)
					json.NewEncoder(w).Encode(response)
				} else {
					w.WriteHeader(http.StatusMethodNotAllowed)
				}
			})
		}
	}
}

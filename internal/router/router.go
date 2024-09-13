package router

import (
	"encoding/json"
	"io"
	"log"
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

					if mock.PrintRequestBody {
						if b, err := io.ReadAll(r.Body); err == nil {
							log.Printf("%s %s\n%s", mock.Verb, r.RequestURI, string(b))
						} else {
							log.Printf("%s %s. Unable to print mock body: %s.", mock.Verb, r.RequestURI, err)
						}
					} else {
						log.Printf("%s %s", mock.Verb, r.RequestURI)
					}
				} else {
					w.WriteHeader(http.StatusMethodNotAllowed)
				}
			})
		}
	}
}

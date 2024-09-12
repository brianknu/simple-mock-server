package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type Message struct {
	Content string `json:"content"`
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		response := Message{Content: "Hello, World!"}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	} else {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func main() {
	http.HandleFunc("/hello", helloHandler)
	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Could not start server: %s", err)
	}
}

package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"simple-mock-server/internal/mock"
	"simple-mock-server/internal/router"
)

func main() {
	portFlag := flag.String("port", "8081", "Server port")
	flag.Parse()
	port := ":" + *portFlag
	mocks := mock.LoadMocksFromFS()
	router.RegisterMocks(mocks)
	fmt.Printf("Starting server on %s.", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Could not start server: %s", err)
	}
}

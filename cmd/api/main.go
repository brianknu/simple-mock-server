package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"simple-mock-server/internal/mock"
	"simple-mock-server/internal/router"
	"simple-mock-server/internal/utils"
)

func main() {
	portFlag := flag.Int("p", 8081, "server port")
	directoryFlag := flag.String("d", "", "directory with the mocks")
	flag.Parse()
	directory, err := utils.ResolveDirectoryWithMocks(*directoryFlag)
	if err != nil {
		log.Fatalf("Could not start simple-mock-server: %s", err)
	}
	port := fmt.Sprintf(":%d", *portFlag)
	mocks, err := mock.LoadMocksFromFS(directory)
	if err != nil {
		log.Fatalf("Could not start simple-mock-server: %s", err)
	}
	router.RegisterMocks(mocks)
	log.Printf("Starting server on %s.", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Could not start simple-mock-server: %s", err)
	}
}
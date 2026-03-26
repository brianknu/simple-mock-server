package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"simple-mock-server/internal/mock"
	"simple-mock-server/internal/router"
	"simple-mock-server/internal/server"
	"simple-mock-server/internal/tui"
	"simple-mock-server/internal/utils"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	portFlag := flag.Int("p", 8081, "server port")
	directoryFlag := flag.String("d", "", "directory with the mocks")
	noTUI := flag.Bool("no-tui", false, "disable TUI, run in headless mode")
	flag.Parse()

	directory, err := utils.ResolveDirectoryWithMocks(*directoryFlag)
	if err != nil {
		log.Fatalf("Could not start simple-mock-server: %s", err)
	}

	if *noTUI {
		runHeadless(*portFlag, directory)
	} else {
		runWithTUI(*portFlag, directory)
	}
}

func runHeadless(port int, directory string) {
	mocks, err := mock.LoadMocksFromFS(directory)
	if err != nil {
		log.Fatalf("Could not start simple-mock-server: %s", err)
	}
	mux := router.NewDynamicMux()
	router.RegisterMocks(mux, mocks, nil)
	addr := fmt.Sprintf(":%d", port)
	log.Printf("Starting server on %s.", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Could not start simple-mock-server: %s", err)
	}
}

func runWithTUI(port int, directory string) {
	srv := server.New(port, directory)
	if err := srv.LoadAndRegister(); err != nil {
		log.Fatalf("Could not start simple-mock-server: %s", err)
	}
	srv.Start()

	model := tui.NewModel(srv)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("TUI error: %s", err)
	}

	srv.Stop()
}

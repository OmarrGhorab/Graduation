package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/OmarrGhorab/courses-attendance-service/internal/bootstrap"
)

func main() {
	container, err := bootstrap.New()
	if err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		if err := container.Shutdown(); err != nil {
			log.Printf("Error during shutdown: %v", err)
		}
		os.Exit(0)
	}()

	if err := container.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

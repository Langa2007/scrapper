package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"scrapper/internal/api"
	"scrapper/internal/crawler"
	"scrapper/internal/service"
)

func main() {
	port := os.Getenv("GO_SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	pyBaseURL := os.Getenv("PYTHON_AI_URL")
	if pyBaseURL == "" {
		pyBaseURL = "http://127.0.0.1:5000"
	}

	log.Printf("Starting scraper server on port %s (python service: %s)", port, pyBaseURL)

	engine := crawler.NewEngine()
	aiClient := service.NewAIClient(pyBaseURL)
	handler := api.NewHandler(engine, aiClient)
	router := api.SetupRouter(handler)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Server listening on http://localhost:%s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server listen failed: %v", err)
		}
	}()

	<-stopChan
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped.")
}

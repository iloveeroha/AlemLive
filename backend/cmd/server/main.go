package main

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/iloveeroha/AlemLive/backend/internal/config"
	"github.com/iloveeroha/AlemLive/backend/internal/httpapi"
)

func main() {
	cfg := config.Load()

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpapi.NewServer(cfg),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("AlemLive backend listening on :%s", cfg.Port)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server stopped: %v", err)
	}
}

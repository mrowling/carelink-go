package server

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/mrowling/carelink-go/internal/database"
	"github.com/mrowling/carelink-go/internal/logger"
)

type Server struct {
	db         *database.DB
	httpServer *http.Server
	lastFetch  time.Time
}

func New(db *database.DB) *Server {
	port := os.Getenv("CARELINK_PORT")
	if port == "" {
		port = "8080"
	}

	s := &Server{
		db: db,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/latest", s.handleLatest)
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/", s.handleRoot)

	s.httpServer = &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	return s
}

func (s *Server) Start() error {
	logger.Info("Server", "Starting HTTP server on %s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	logger.Info("Server", "Shutting down HTTP server")
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) UpdateLastFetch(t time.Time) {
	s.lastFetch = t
}

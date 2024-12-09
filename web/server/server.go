package server

import (
    "context"
    "net/http"
    "time"

    "github.com/gosom/google-maps-scraper/web/handlers"
    "go.uber.org/zap"
)

type Server struct {
    srv    *http.Server
    logger *zap.Logger
}

func New(handler *handlers.JobHandler, logger *zap.Logger) *Server {
    mux := http.NewServeMux()

    // Register routes
    mux.HandleFunc("/api/jobs", handler.CreateJob)

    srv := &http.Server{
        Addr:         ":6060",
        Handler:      mux,
        ReadTimeout:  30 * time.Second,
        WriteTimeout: 30 * time.Second,
        IdleTimeout:  120 * time.Second,
    }

    return &Server{
        srv:    srv,
        logger: logger,
    }
}

func (s *Server) Start() error {
    s.logger.Info("API server is running",
        zap.String("url", "http://localhost"+s.srv.Addr),
        zap.String("port", s.srv.Addr),
    )
    return s.srv.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
    s.logger.Info("shutting down server")
    return s.srv.Shutdown(ctx)
} 
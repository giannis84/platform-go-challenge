package internal

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/giannis84/platform-go-challenge/internal/logging"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// RoutesRegistry is a function that registers routes on a chi.Router
type RoutesRegistry func(r chi.Router)

// ServiceConfig holds configuration for creating a service
type ServiceConfig struct {
	Addr           string
	Logger         *slog.Logger
	Routes 		   RoutesRegistry
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	IdleTimeout    time.Duration
}

// Service wraps an HTTP server with its router
type Service struct {
	HTTPServer *http.Server
	Router     *chi.Mux
	Logger     *slog.Logger
}

// NewService creates a new service with the given configuration
func NewService(cfg ServiceConfig) *Service {
	router := chi.NewRouter()

	// Initialize common middleware
	router.Use(middleware.RequestID)
	router.Use(logging.RequestLogger(cfg.Logger))
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	// Register routes
	if cfg.Routes != nil {
		cfg.Routes(router)
	}

	// Apply default timeouts if not provided in config
	readTimeout := cfg.ReadTimeout
	if readTimeout == 0 {
		readTimeout = 15 * time.Second
	}
	writeTimeout := cfg.WriteTimeout
	if writeTimeout == 0 {
		writeTimeout = 15 * time.Second
	}
	idleTimeout := cfg.IdleTimeout
	if idleTimeout == 0 {
		idleTimeout = 60 * time.Second
	}

	httpServer := &http.Server{
		Addr:         cfg.Addr,
		Handler:      router,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	return &Service{
		HTTPServer: httpServer,
		Router:     router,
		Logger:     cfg.Logger,
	}
}

// ListenAndServeWrapper starts the http service
func (s *Service) ListenAndServeWrapper(service string) error {
	s.Logger.Info("starting http service", service, slog.String("port", s.HTTPServer.Addr))
	return s.HTTPServer.ListenAndServe()
}
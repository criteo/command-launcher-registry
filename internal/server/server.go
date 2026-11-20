package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/criteo/command-launcher-registry/internal/auth"
	"github.com/criteo/command-launcher-registry/internal/config"
	"github.com/criteo/command-launcher-registry/internal/server/handlers"
	"github.com/criteo/command-launcher-registry/internal/server/middleware"
	"github.com/criteo/command-launcher-registry/internal/storage"
)

// Server represents the HTTP server
type Server struct {
	config        *config.Config
	logger        *slog.Logger
	store         storage.Store
	authenticator auth.Authenticator
	httpServer    *http.Server
}

// NewServer creates a new server instance
func NewServer(cfg *config.Config, logger *slog.Logger, store storage.Store, authenticator auth.Authenticator) *Server {
	return &Server{
		config:        cfg,
		logger:        logger,
		store:         store,
		authenticator: authenticator,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	// Build the application handler
	router := NewHandler(s.config, s.logger, s.store, s.authenticator)

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second, // Must be longer than OCI push timeout (60s)
		IdleTimeout:  120 * time.Second,
	}

	// Log server start
	s.logger.Info("Starting server",
		"host", s.config.Server.Host,
		"port", s.config.Server.Port,
		"storage_uri", s.config.Storage.URI,
		"auth_type", s.config.Auth.Type)

	// Start server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Setup graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal or server error
	select {
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	case sig := <-quit:
		s.logger.Info("Shutdown signal received", "signal", sig.String())
		return s.Shutdown()
	}
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	s.logger.Info("Initiating graceful shutdown")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error("Server shutdown failed", "error", err)
		return err
	}

	// Close storage
	if err := s.store.Close(); err != nil {
		s.logger.Error("Storage close failed", "error", err)
		return err
	}

	s.logger.Info("Server stopped gracefully")
	return nil
}

// NewHandler returns the complete application HTTP handler: router,
// middleware and every API handler, wired to the given store and
// authenticator. Start serves it; tests can drive it in-process
// (see the servertest package). cfg carries the settings that shape the
// wiring: the server-admin list, in force only when cfg.RolesEnforced().
func NewHandler(cfg *config.Config, logger *slog.Logger, store storage.Store, authenticator auth.Authenticator) http.Handler {
	indexHandler := handlers.NewIndexHandler(store, logger)
	authorizer := handlers.NewAuthorizer(serverAdmins(cfg), logger)
	registryHandler := handlers.NewRegistryHandler(store, authorizer, logger)
	packageHandler := handlers.NewPackageHandler(store, logger)
	versionHandler := handlers.NewVersionHandler(store, logger)
	healthHandler := handlers.NewHealthHandler(store, logger)
	metricsHandler := handlers.NewMetricsHandler(logger)
	whoamiHandler := handlers.NewWhoamiHandler(authenticator, serverAdmins(cfg), logger)

	requireAuth := middleware.RequireAuth(authenticator)

	router := chi.NewRouter()

	// Global middleware (applied to all routes)
	router.Use(middleware.Logging(logger))
	router.Use(middleware.NewRateLimiter(100)) // 100 req/min per IP
	router.Use(middleware.CORS())

	// API v1 routes
	router.Route("/api/v1", func(r chi.Router) {
		// Health and metrics endpoints (no auth required)
		r.Get("/health", healthHandler.GetHealth)
		r.Get("/metrics", metricsHandler.GetMetrics)

		// Whoami endpoint (the handler authenticates the caller itself)
		r.Get("/whoami", whoamiHandler.GetWhoami)

		// Registry index endpoint (no auth required for GET)
		r.Get("/registry/{name}/index.json", indexHandler.GetIndex)
		r.Options("/registry/{name}/index.json", indexHandler.HandleOptions)

		// Registry endpoints. RequireAuth only gates POST, PUT and DELETE.
		r.Route("/registry", func(r chi.Router) {
			r.With(requireAuth).Get("/", registryHandler.ListRegistries)
			r.With(requireAuth).Post("/", registryHandler.CreateRegistry)

			// Single registry operations
			r.Route("/{name}", func(r chi.Router) {
				r.Get("/", registryHandler.GetRegistry)
				r.With(requireAuth).Put("/", registryHandler.UpdateRegistry)
				r.With(requireAuth).Delete("/", registryHandler.DeleteRegistry)

				// Package endpoints
				r.Route("/package", func(r chi.Router) {
					r.Get("/", packageHandler.ListPackages)
					r.With(requireAuth).Post("/", packageHandler.CreatePackage)

					// Single package operations
					r.Route("/{package}", func(r chi.Router) {
						r.Get("/", packageHandler.GetPackage)
						r.With(requireAuth).Put("/", packageHandler.UpdatePackage)
						r.With(requireAuth).Delete("/", packageHandler.DeletePackage)

						// Version endpoints
						r.Route("/version", func(r chi.Router) {
							r.Get("/", versionHandler.ListVersions)
							r.With(requireAuth).Post("/", versionHandler.CreateVersion)

							// Single version operations
							r.Route("/{version}", func(r chi.Router) {
								r.Get("/", versionHandler.GetVersion)
								r.With(requireAuth).Delete("/", versionHandler.DeleteVersion)
							})
						})
					})
				})
			})
		})
	})

	return router
}

// serverAdmins returns the server-admin list in force: cfg.Admins when role
// enforcement is on, nil otherwise.
func serverAdmins(cfg *config.Config) []string {
	if !cfg.RolesEnforced() {
		return nil
	}
	return cfg.Admins
}

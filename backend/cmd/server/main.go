package main

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"tv/internal/config"
	"tv/internal/handler"
	"tv/internal/middleware"
	"tv/internal/store"
	"tv/internal/ws"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("configuration error: %v", err)
	}

	hub := ws.NewHub()
	go hub.Run()
	defer hub.Stop()

	srv := &http.Server{
		Addr:              ":" + strconv.Itoa(cfg.Port),
		Handler:           buildRouter(cfg, hub),
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
	}

	// Start accepting connections in a background goroutine so the main goroutine
	// can block on the shutdown signal channel.
	go func() {
		log.Printf("server listening on :%d (frontend origin: %s)", cfg.Port, cfg.FrontendURL)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Block until SIGINT (Ctrl-C) or SIGTERM (Docker stop / systemd).
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	log.Println("shutdown signal received — draining in-flight requests (timeout: 15s)...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown error: %v", err)
	} else {
		log.Println("server stopped cleanly")
	}
}

// buildRouter wires all routes and the middleware stack.
// It is a separate function so it can be reused in tests without starting a real listener.
func buildRouter(cfg *config.Config, hub *ws.Hub) http.Handler {
	if hub == nil {
		hub = ws.NewHub()
	}

	mux := http.NewServeMux()

	// ── Stores ────────────────────────────────────────────────────────────────
	sceneStore := store.NewSceneStore()
	userStore := store.NewUserStore()
	settingsStore := store.NewSettingsStore()
	telemetryStore := store.NewTelemetryStore()

	// ── Handlers ──────────────────────────────────────────────────────────────
	startedAt := time.Now().UTC()
	sceneHandler := handler.NewSceneHandler(sceneStore, hub)
	userHandler := handler.NewUserHandler(userStore, hub)
	settingsHandler := handler.NewSettingsHandler(settingsStore, hub)
	dashboardHandler := handler.NewDashboardHandler(sceneStore, userStore, settingsStore, hub, startedAt, "1.0.0", "development")
	telemetryHandler := handler.NewTelemetryHandler(telemetryStore, settingsStore, hub)
	wsHandler := ws.NewHandler(hub, cfg.FrontendURL)

	// ── Auth middleware ────────────────────────────────────────────────────────
	auth := middleware.APIKey(cfg)

	// ── Public routes (no API key required) ───────────────────────────────────
	mux.HandleFunc("GET /health", handler.Health)
	mux.Handle("GET /ws", wsHandler)
	mux.HandleFunc("GET /api/v1/dashboard/status", dashboardHandler.Status)
	mux.HandleFunc("GET /api/v1/dashboard/stats", dashboardHandler.Stats)
	mux.HandleFunc("GET /api/v1/scenes", sceneHandler.List)
	mux.HandleFunc("GET /api/v1/scenes/{id}", sceneHandler.GetByID)
	mux.HandleFunc("GET /api/v1/users", userHandler.List)
	mux.HandleFunc("GET /api/v1/users/{id}", userHandler.GetByID)
	mux.HandleFunc("GET /api/v1/settings", settingsHandler.Get)
	mux.HandleFunc("POST /api/v1/telemetry/events", telemetryHandler.Ingest)

	// ── Protected routes (API key required) ───────────────────────────────────
	mux.Handle("POST /api/v1/scenes", auth(http.HandlerFunc(sceneHandler.Create)))
	mux.Handle("PUT /api/v1/scenes/{id}", auth(http.HandlerFunc(sceneHandler.Update)))
	mux.Handle("DELETE /api/v1/scenes/{id}", auth(http.HandlerFunc(sceneHandler.Delete)))
	mux.Handle("POST /api/v1/users", auth(http.HandlerFunc(userHandler.Create)))
	mux.Handle("PUT /api/v1/users/{id}", auth(http.HandlerFunc(userHandler.Update)))
	mux.Handle("DELETE /api/v1/users/{id}", auth(http.HandlerFunc(userHandler.Delete)))
	mux.Handle("PUT /api/v1/settings", auth(http.HandlerFunc(settingsHandler.Update)))
	mux.Handle("POST /api/v1/settings/reset", auth(http.HandlerFunc(settingsHandler.Reset)))
	mux.Handle("GET /api/v1/telemetry/events", auth(http.HandlerFunc(telemetryHandler.List)))
	mux.Handle("GET /api/v1/telemetry/summary", auth(http.HandlerFunc(telemetryHandler.Summary)))

	// ── Global middleware stack ────────────────────────────────────────────────
	// Request flow: Logger → Recover → CORS → mux
	// CORS is innermost so it handles OPTIONS pre-flights before our handlers run.
	corsMW := middleware.Cors(cfg)
	return middleware.Chain(
		corsMW.Handler(mux),
		middleware.Logger,
		middleware.Recover,
	)
}

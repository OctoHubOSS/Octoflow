package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/infinitybotlist/eureka/zapchi"
	"go.uber.org/zap"

	"github.com/git-logs/client/webserver/ontos"
	"github.com/git-logs/client/webserver/state"
)

func main() {
	state.Setup()
	defer state.Close()

	r := chi.NewRouter()

	r.Use(zapchi.Logger(state.Logger.Sugar().Named("zapchi"), "api"))
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.Timeout(60 * time.Second))

	r.HandleFunc("/", ontos.IndexPage)

	// API
	r.HandleFunc("/api/counts", ontos.ApiStats)
	r.HandleFunc("/api/events/listview", ontos.ApiEventsListView)
	r.HandleFunc("/api/events/csview", ontos.ApiEventsCommaSepView)
	r.HandleFunc("/health", ontos.HealthCheck)

	// KittyCat (webhook route)
	r.Get("/kittycat", ontos.GetWebhookRoute)
	r.Post("/kittycat", ontos.HandleWebhookRoute)
	r.HandleFunc("/audit", ontos.AuditEvent)

	srv := &http.Server{
		Addr:    state.Config.Port,
		Handler: r,
	}

	// Graceful shutdown channel
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		state.Logger.Info("Starting webserver on " + state.Config.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			state.Logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	<-done
	state.Logger.Info("Webserver is shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		state.Logger.Fatal("Server shutdown failed", zap.Error(err))
	}

	state.Logger.Info("Webserver gracefully stopped")
}

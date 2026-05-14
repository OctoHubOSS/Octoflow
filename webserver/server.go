package main

import (
	"net/http"
	"time"

	"github.com/git-logs/client/webserver/ontos"
	"github.com/git-logs/client/webserver/state"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/infinitybotlist/eureka/zapchi"
)

func main() {
	state.Setup()

	defer state.Close()

	r := chi.NewMux()

	r.Use(zapchi.Logger(state.Logger.Sugar().Named("zapchi"), "api"), middleware.Recoverer, middleware.RealIP, middleware.RequestID, middleware.Timeout(60*time.Second))

	// Webhook route
	r.Get("/kittycat", ontos.GetWebhookRoute)
	r.Post("/kittycat", ontos.HandleWebhookRoute)
	r.HandleFunc("/", ontos.IndexPage)
	r.HandleFunc("/audit", ontos.AuditEvent)
	r.HandleFunc("/health", ontos.HealthCheck)

	// API
	r.HandleFunc("/api/counts", ontos.ApiStats)
	r.HandleFunc("/api/events/listview", ontos.ApiEventsListView)

	r.HandleFunc("/api/events/csview", ontos.ApiEventsCommaSepView)

	http.ListenAndServe(state.Config.Port, r)
}

package main

import (
	"net/http"
	"time"

	"github.com/OctoHubOSS/Octoflow/webserver/ontos"
	"github.com/OctoHubOSS/Octoflow/webserver/state"

	"github.com/PlexiOSS/Keel/zapchi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	state.Setup()

	defer state.Close()

	go ontos.StartStatusSnapshotter()

	r := chi.NewMux()

	r.Use(zapchi.Logger(state.Logger.Sugar().Named("zapchi"), "api"), middleware.Recoverer, middleware.RealIP, middleware.RequestID, middleware.Timeout(60*time.Second))

	r.Get("/kittycat", ontos.GetWebhookRoute)
	r.Post("/kittycat", ontos.HandleWebhookRoute)
	r.HandleFunc("/", ontos.IndexPage)
	r.HandleFunc("/audit", ontos.AuditEvent)

	r.HandleFunc("/api/counts", ontos.ApiStats)
	r.HandleFunc("/api/events/listview", ontos.ApiEventsListView)
	r.HandleFunc("/api/events/csview", ontos.ApiEventsCommaSepView)
	r.HandleFunc("/api/health", ontos.ApiHealth)
	r.HandleFunc("/api/status/history", ontos.ApiStatusHistory)

	http.ListenAndServe(state.Config.Port, r)
}

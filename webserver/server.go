//  Copyright (C) 2026 NodeByte LTD

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
	r.HandleFunc("/api/stats/summary", ontos.ApiPublicStats)

	r.Route("/api/dashboard", func(r chi.Router) {
		r.Use(ontos.DashboardAuth)

		r.Get("/guilds/{guildId}", ontos.ApiDashboardGuild)
		r.Get("/guilds/{guildId}/channels", ontos.ApiDashboardChannels)
		r.Get("/guilds/{guildId}/analytics", ontos.ApiDashboardAnalytics)
		r.Post("/guilds/{guildId}/webhooks", ontos.ApiDashboardCreateWebhook)

		r.Patch("/webhooks/{id}", ontos.ApiDashboardUpdateWebhook)
		r.Post("/webhooks/{id}/reset-secret", ontos.ApiDashboardResetWebhookSecret)
		r.Delete("/webhooks/{id}", ontos.ApiDashboardDeleteWebhook)
		r.Post("/webhooks/{webhookId}/repos", ontos.ApiDashboardCreateRepo)
		r.Post("/webhooks/{webhookId}/modifiers", ontos.ApiDashboardCreateModifier)

		r.Patch("/repos/{id}", ontos.ApiDashboardUpdateRepo)
		r.Delete("/repos/{id}", ontos.ApiDashboardDeleteRepo)

		r.Patch("/modifiers/{id}", ontos.ApiDashboardUpdateModifier)
		r.Delete("/modifiers/{id}", ontos.ApiDashboardDeleteModifier)
	})

	r.Route("/api/admin", func(r chi.Router) {
		r.Use(ontos.DashboardAuth, ontos.AdminAuth)

		r.Get("/whoami", ontos.ApiAdminWhoAmI)
		r.Get("/stats", ontos.ApiAdminStats)
		r.Get("/guilds", ontos.ApiAdminGuilds)
		r.Post("/guilds/{guildId}/ban", ontos.ApiAdminBanGuild)
		r.Get("/logs", ontos.ApiAdminLogs)
		r.Get("/audit-log", ontos.ApiAdminAuditLog)
	})

	http.ListenAndServe(state.Config.Port, r)
}

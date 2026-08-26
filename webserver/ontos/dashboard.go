package ontos

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/OctoHubOSS/Octoflow/webserver/logos/eventmodifiers"
	"github.com/OctoHubOSS/Octoflow/webserver/state"
	"github.com/go-chi/chi/v5"
)

func DashboardAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		secret := state.Config.DashboardInternalSecret

		if secret == "" || r.Header.Get("X-Internal-Secret") != secret {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized"))
			return
		}

		next.ServeHTTP(w, r)
	})
}

type dashboardRepo struct {
	ID        string `json:"id"`
	RepoName  string `json:"repo_name"`
	ChannelID string `json:"channel_id"`
}

type dashboardModifier struct {
	ID              string   `json:"id"`
	RepoID          string   `json:"repo_id,omitempty"`
	Events          []string `json:"events"`
	Blacklisted     bool     `json:"blacklisted"`
	Whitelisted     bool     `json:"whitelisted"`
	RedirectChannel string   `json:"redirect_channel,omitempty"`
	Priority        int      `json:"priority"`
}

type dashboardWebhook struct {
	ID             string              `json:"id"`
	Comment        string              `json:"comment"`
	Broken         bool                `json:"broken"`
	CreatedAt      time.Time           `json:"created_at"`
	Repos          []dashboardRepo     `json:"repos"`
	EventModifiers []dashboardModifier `json:"event_modifiers"`
}

func ApiDashboardGuild(w http.ResponseWriter, r *http.Request) {
	guildId := chi.URLParam(r, "guildId")

	if guildId == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Missing guildId"))
		return
	}

	rows, err := state.Pool.Query(
		state.Context,
		"SELECT id, comment, broken, created_at FROM "+state.TableWebhooks+" WHERE guild_id = $1",
		guildId,
	)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error fetching webhooks: " + err.Error()))
		return
	}
	defer rows.Close()

	webhooks := []dashboardWebhook{}

	for rows.Next() {
		var wh dashboardWebhook

		if err := rows.Scan(&wh.ID, &wh.Comment, &wh.Broken, &wh.CreatedAt); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Error scanning webhook: " + err.Error()))
			return
		}

		wh.Repos = []dashboardRepo{}
		wh.EventModifiers = []dashboardModifier{}

		webhooks = append(webhooks, wh)
	}

	for i := range webhooks {
		repoRows, err := state.Pool.Query(
			state.Context,
			"SELECT id, repo_name, channel_id FROM "+state.TableRepos+" WHERE webhook_id = $1",
			webhooks[i].ID,
		)

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Error fetching repos: " + err.Error()))
			return
		}

		for repoRows.Next() {
			var repo dashboardRepo

			if err := repoRows.Scan(&repo.ID, &repo.RepoName, &repo.ChannelID); err != nil {
				repoRows.Close()
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Error scanning repo: " + err.Error()))
				return
			}

			webhooks[i].Repos = append(webhooks[i].Repos, repo)
		}
		repoRows.Close()

		modifiers, err := eventmodifiers.GetEventModifiers(webhooks[i].ID, "")

		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Error fetching event modifiers: " + err.Error()))
			return
		}

		for _, m := range modifiers {
			webhooks[i].EventModifiers = append(webhooks[i].EventModifiers, dashboardModifier{
				ID:              m.ID,
				RepoID:          m.RepoID,
				Events:          m.Events,
				Blacklisted:     m.Blacklisted,
				Whitelisted:     m.Whitelisted,
				RedirectChannel: m.RedirectChannel,
				Priority:        m.Priority,
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{"webhooks": webhooks})
}

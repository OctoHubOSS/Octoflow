//  Copyright (C) 2026 NodeByte LTD

package ontos

import (
	"crypto/rand"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/OctoHubOSS/Octoflow/webserver/logos/eventmodifiers"
	"github.com/OctoHubOSS/Octoflow/webserver/state"
	"github.com/go-chi/chi/v5"
)

const (
	maxWebhooksPerGuild    = 5
	maxModifiersPerWebhook = 10
)

const alphanumeric = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func randomAlphanumeric(n int) (string, error) {
	b := make([]byte, n)
	for i := range b {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphanumeric))))
		if err != nil {
			return "", err
		}
		b[i] = alphanumeric[idx.Int64()]
	}
	return string(b), nil
}

func normalizeEvents(events string) []string {
	cleaned := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(events, "`", ""), ",", " "), "  ", " "))
	return strings.Split(cleaned, " ")
}

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

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func decodeBody(w http.ResponseWriter, r *http.Request, v any) bool {
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Could not read request body")
		return false
	}
	if len(body) > 0 {
		if err := json.Unmarshal(body, v); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid JSON body: "+err.Error())
			return false
		}
	}
	return true
}

func resolveChannelNames(guildId string) map[string]string {
	names := map[string]string{}

	channels, err := state.Discord.GuildChannels(guildId)
	if err != nil {
		return names
	}

	for _, c := range channels {
		names[c.ID] = c.Name
	}

	return names
}

type dashboardRepo struct {
	ID          string `json:"id"`
	RepoName    string `json:"repo_name"`
	ChannelID   string `json:"channel_id"`
	ChannelName string `json:"channel_name,omitempty"`
	UseThreads  bool   `json:"use_threads"`
}

type dashboardModifier struct {
	ID                  string   `json:"id"`
	RepoID              string   `json:"repo_id,omitempty"`
	Events              []string `json:"events"`
	Blacklisted         bool     `json:"blacklisted"`
	Whitelisted         bool     `json:"whitelisted"`
	RedirectChannel     string   `json:"redirect_channel,omitempty"`
	RedirectChannelName string   `json:"redirect_channel_name,omitempty"`
	Priority            int      `json:"priority"`
}

type dashboardWebhook struct {
	ID             string              `json:"id"`
	Comment        string              `json:"comment"`
	Broken         bool                `json:"broken"`
	BatchEvents    bool                `json:"batch_events"`
	CreatedAt      time.Time           `json:"created_at"`
	Repos          []dashboardRepo     `json:"repos"`
	EventModifiers []dashboardModifier `json:"event_modifiers"`
}

func ApiDashboardGuild(w http.ResponseWriter, r *http.Request) {
	guildId := chi.URLParam(r, "guildId")

	if guildId == "" {
		writeError(w, http.StatusBadRequest, "Missing guildId")
		return
	}

	channelNames := resolveChannelNames(guildId)

	rows, err := state.Pool.Query(
		state.Context,
		"SELECT id, comment, broken, batch_events, created_at FROM "+state.TableWebhooks+" WHERE guild_id = $1",
		guildId,
	)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error fetching webhooks: "+err.Error())
		return
	}
	defer rows.Close()

	webhooks := []dashboardWebhook{}

	for rows.Next() {
		var wh dashboardWebhook

		if err := rows.Scan(&wh.ID, &wh.Comment, &wh.Broken, &wh.BatchEvents, &wh.CreatedAt); err != nil {
			writeError(w, http.StatusInternalServerError, "Error scanning webhook: "+err.Error())
			return
		}

		wh.Repos = []dashboardRepo{}
		wh.EventModifiers = []dashboardModifier{}

		webhooks = append(webhooks, wh)
	}

	for i := range webhooks {
		repoRows, err := state.Pool.Query(
			state.Context,
			"SELECT id, repo_name, channel_id, use_threads FROM "+state.TableRepos+" WHERE webhook_id = $1",
			webhooks[i].ID,
		)

		if err != nil {
			writeError(w, http.StatusInternalServerError, "Error fetching repos: "+err.Error())
			return
		}

		for repoRows.Next() {
			var repo dashboardRepo

			if err := repoRows.Scan(&repo.ID, &repo.RepoName, &repo.ChannelID, &repo.UseThreads); err != nil {
				repoRows.Close()
				writeError(w, http.StatusInternalServerError, "Error scanning repo: "+err.Error())
				return
			}

			repo.ChannelName = channelNames[repo.ChannelID]
			webhooks[i].Repos = append(webhooks[i].Repos, repo)
		}
		repoRows.Close()

		modifiers, err := eventmodifiers.GetEventModifiers(webhooks[i].ID, "")

		if err != nil {
			writeError(w, http.StatusInternalServerError, "Error fetching event modifiers: "+err.Error())
			return
		}

		for _, m := range modifiers {
			webhooks[i].EventModifiers = append(webhooks[i].EventModifiers, dashboardModifier{
				ID:                  m.ID,
				RepoID:              m.RepoID,
				Events:              m.Events,
				Blacklisted:         m.Blacklisted,
				Whitelisted:         m.Whitelisted,
				RedirectChannel:     m.RedirectChannel,
				RedirectChannelName: channelNames[m.RedirectChannel],
				Priority:            m.Priority,
			})
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"webhooks": webhooks})
}

func ApiDashboardChannels(w http.ResponseWriter, r *http.Request) {
	guildId := chi.URLParam(r, "guildId")

	channels, err := state.Discord.GuildChannels(guildId)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error fetching channels: "+err.Error())
		return
	}

	type channel struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Type int    `json:"type"`
	}

	out := []channel{}
	for _, c := range channels {
		if c.Type == 0 || c.Type == 5 {
			out = append(out, channel{ID: c.ID, Name: c.Name, Type: int(c.Type)})
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"channels": out})
}

type createWebhookRequest struct {
	Comment      string `json:"comment"`
	Broken       bool   `json:"broken"`
	ActingUserID string `json:"acting_user_id"`
}

func ApiDashboardCreateWebhook(w http.ResponseWriter, r *http.Request) {
	guildId := chi.URLParam(r, "guildId")

	var req createWebhookRequest
	if !decodeBody(w, r, &req) {
		return
	}

	if req.Comment == "" || req.ActingUserID == "" {
		writeError(w, http.StatusBadRequest, "comment and acting_user_id are required")
		return
	}

	var count int
	err := state.Pool.QueryRow(state.Context, "SELECT COUNT(1) FROM "+state.TableWebhooks+" WHERE guild_id = $1", guildId).Scan(&count)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error checking webhook count: "+err.Error())
		return
	}
	if count >= maxWebhooksPerGuild {
		writeError(w, http.StatusBadRequest, "You can't have more than 5 webhooks per guild")
		return
	}

	// Ensure the guild row exists (mirrors every Rust command's guild bootstrap check).
	_, err = state.Pool.Exec(state.Context, "INSERT INTO "+state.TableGuilds+" (id) VALUES ($1) ON CONFLICT (id) DO NOTHING", guildId)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error ensuring guild row: "+err.Error())
		return
	}

	id, err := randomAlphanumeric(32)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error generating webhook id")
		return
	}
	secret, err := randomAlphanumeric(256)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error generating webhook secret")
		return
	}

	_, err = state.Pool.Exec(
		state.Context,
		"INSERT INTO "+state.TableWebhooks+" (id, guild_id, comment, secret, broken, created_by, last_updated_by) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		id, guildId, req.Comment, secret, req.Broken, req.ActingUserID, req.ActingUserID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error creating webhook: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"id":     id,
		"secret": secret,
		"url":    state.Config.APIUrl + "/kittycat?id=" + id,
	})
}

type updateWebhookRequest struct {
	GuildID      string  `json:"guild_id"`
	Comment      *string `json:"comment"`
	Broken       *bool   `json:"broken"`
	BatchEvents  *bool   `json:"batch_events"`
	ActingUserID string  `json:"acting_user_id"`
}

func ApiDashboardUpdateWebhook(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateWebhookRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if req.GuildID == "" || req.ActingUserID == "" {
		writeError(w, http.StatusBadRequest, "guild_id and acting_user_id are required")
		return
	}

	tx, err := state.Pool.Begin(state.Context)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error starting transaction: "+err.Error())
		return
	}
	defer tx.Rollback(state.Context)

	if req.Comment != nil {
		if _, err := tx.Exec(state.Context, "UPDATE "+state.TableWebhooks+" SET comment = $1 WHERE id = $2 AND guild_id = $3", *req.Comment, id, req.GuildID); err != nil {
			writeError(w, http.StatusInternalServerError, "Error updating comment: "+err.Error())
			return
		}
	}
	if req.Broken != nil {
		if _, err := tx.Exec(state.Context, "UPDATE "+state.TableWebhooks+" SET broken = $1 WHERE id = $2 AND guild_id = $3", *req.Broken, id, req.GuildID); err != nil {
			writeError(w, http.StatusInternalServerError, "Error updating broken flag: "+err.Error())
			return
		}
	}
	if req.BatchEvents != nil {
		if _, err := tx.Exec(state.Context, "UPDATE "+state.TableWebhooks+" SET batch_events = $1 WHERE id = $2 AND guild_id = $3", *req.BatchEvents, id, req.GuildID); err != nil {
			writeError(w, http.StatusInternalServerError, "Error updating batch_events flag: "+err.Error())
			return
		}
	}

	if _, err := tx.Exec(state.Context, "UPDATE "+state.TableWebhooks+" SET last_updated_at = NOW(), last_updated_by = $1 WHERE id = $2 AND guild_id = $3", req.ActingUserID, id, req.GuildID); err != nil {
		writeError(w, http.StatusInternalServerError, "Error stamping last_updated_by: "+err.Error())
		return
	}

	if err := tx.Commit(state.Context); err != nil {
		writeError(w, http.StatusInternalServerError, "Error committing transaction: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type resetWebhookSecretRequest struct {
	GuildID      string `json:"guild_id"`
	ActingUserID string `json:"acting_user_id"`
}

func ApiDashboardResetWebhookSecret(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req resetWebhookSecretRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if req.GuildID == "" || req.ActingUserID == "" {
		writeError(w, http.StatusBadRequest, "guild_id and acting_user_id are required")
		return
	}

	secret, err := randomAlphanumeric(256)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error generating secret")
		return
	}

	result, err := state.Pool.Exec(
		state.Context,
		"UPDATE "+state.TableWebhooks+" SET secret = $1, last_updated_at = NOW(), last_updated_by = $2 WHERE id = $3 AND guild_id = $4",
		secret, req.ActingUserID, id, req.GuildID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error resetting secret: "+err.Error())
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "Webhook not found")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"secret": secret})
}

type deleteRequest struct {
	GuildID string `json:"guild_id"`
}

func ApiDashboardDeleteWebhook(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req deleteRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if req.GuildID == "" {
		writeError(w, http.StatusBadRequest, "guild_id is required")
		return
	}

	if _, err := state.Pool.Exec(state.Context, "DELETE FROM "+state.TableWebhooks+" WHERE id = $1 AND guild_id = $2", id, req.GuildID); err != nil {
		writeError(w, http.StatusInternalServerError, "Error deleting webhook: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// --- Repos ---

type createRepoRequest struct {
	GuildID      string `json:"guild_id"`
	Owner        string `json:"owner"`
	Name         string `json:"name"`
	ChannelID    string `json:"channel_id"`
	ActingUserID string `json:"acting_user_id"`
}

func ApiDashboardCreateRepo(w http.ResponseWriter, r *http.Request) {
	webhookId := chi.URLParam(r, "webhookId")

	var req createRepoRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if req.GuildID == "" || req.Owner == "" || req.Name == "" || req.ChannelID == "" || req.ActingUserID == "" {
		writeError(w, http.StatusBadRequest, "guild_id, owner, name, channel_id, and acting_user_id are required")
		return
	}

	var webhookCount int
	err := state.Pool.QueryRow(state.Context, "SELECT COUNT(1) FROM "+state.TableWebhooks+" WHERE id = $1 AND guild_id = $2", webhookId, req.GuildID).Scan(&webhookCount)
	if err != nil || webhookCount == 0 {
		writeError(w, http.StatusNotFound, "That webhook doesn't exist")
		return
	}

	repoName := strings.ToLower(req.Owner + "/" + req.Name)

	var existing int
	err = state.Pool.QueryRow(state.Context, "SELECT COUNT(1) FROM "+state.TableRepos+" WHERE lower(repo_name) = $1 AND webhook_id = $2", repoName, webhookId).Scan(&existing)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error checking for existing repo: "+err.Error())
		return
	}
	if existing > 0 {
		writeError(w, http.StatusBadRequest, "That repo already exists on this webhook")
		return
	}

	id, err := randomAlphanumeric(32)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error generating repo id")
		return
	}

	_, err = state.Pool.Exec(
		state.Context,
		"INSERT INTO "+state.TableRepos+" (id, webhook_id, repo_name, channel_id, guild_id, created_by, last_updated_by) VALUES ($1, $2, $3, $4, $5, $6, $7)",
		id, webhookId, repoName, req.ChannelID, req.GuildID, req.ActingUserID, req.ActingUserID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error creating repo: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"id": id, "repo_name": repoName})
}

type updateRepoRequest struct {
	GuildID      string  `json:"guild_id"`
	RepoName     *string `json:"repo_name"`
	ChannelID    *string `json:"channel_id"`
	UseThreads   *bool   `json:"use_threads"`
	ActingUserID string  `json:"acting_user_id"`
}

func ApiDashboardUpdateRepo(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateRepoRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if req.GuildID == "" || req.ActingUserID == "" {
		writeError(w, http.StatusBadRequest, "guild_id and acting_user_id are required")
		return
	}

	tx, err := state.Pool.Begin(state.Context)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error starting transaction: "+err.Error())
		return
	}
	defer tx.Rollback(state.Context)

	if req.RepoName != nil {
		lowered := strings.ToLower(*req.RepoName)
		if _, err := tx.Exec(state.Context, "UPDATE "+state.TableRepos+" SET repo_name = $1 WHERE id = $2 AND guild_id = $3", lowered, id, req.GuildID); err != nil {
			writeError(w, http.StatusInternalServerError, "Error updating repo name: "+err.Error())
			return
		}
	}
	if req.ChannelID != nil {
		if _, err := tx.Exec(state.Context, "UPDATE "+state.TableRepos+" SET channel_id = $1 WHERE id = $2 AND guild_id = $3", *req.ChannelID, id, req.GuildID); err != nil {
			writeError(w, http.StatusInternalServerError, "Error updating channel: "+err.Error())
			return
		}
	}
	if req.UseThreads != nil {
		if _, err := tx.Exec(state.Context, "UPDATE "+state.TableRepos+" SET use_threads = $1 WHERE id = $2 AND guild_id = $3", *req.UseThreads, id, req.GuildID); err != nil {
			writeError(w, http.StatusInternalServerError, "Error updating use_threads flag: "+err.Error())
			return
		}
	}

	if _, err := tx.Exec(state.Context, "UPDATE "+state.TableRepos+" SET last_updated_at = NOW(), last_updated_by = $1 WHERE id = $2 AND guild_id = $3", req.ActingUserID, id, req.GuildID); err != nil {
		writeError(w, http.StatusInternalServerError, "Error stamping last_updated_by: "+err.Error())
		return
	}

	if err := tx.Commit(state.Context); err != nil {
		writeError(w, http.StatusInternalServerError, "Error committing transaction: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func ApiDashboardDeleteRepo(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req deleteRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if req.GuildID == "" {
		writeError(w, http.StatusBadRequest, "guild_id is required")
		return
	}

	if _, err := state.Pool.Exec(state.Context, "DELETE FROM "+state.TableRepos+" WHERE id = $1 AND guild_id = $2", id, req.GuildID); err != nil {
		writeError(w, http.StatusInternalServerError, "Error deleting repo: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type createModifierRequest struct {
	GuildID         string  `json:"guild_id"`
	Events          string  `json:"events"`
	Blacklisted     bool    `json:"blacklisted"`
	Whitelisted     bool    `json:"whitelisted"`
	Priority        int     `json:"priority"`
	RepoID          *string `json:"repo_id"`
	RedirectChannel *string `json:"redirect_channel"`
	ActingUserID    string  `json:"acting_user_id"`
}

func ApiDashboardCreateModifier(w http.ResponseWriter, r *http.Request) {
	webhookId := chi.URLParam(r, "webhookId")

	var req createModifierRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if req.GuildID == "" || req.Events == "" || req.ActingUserID == "" {
		writeError(w, http.StatusBadRequest, "guild_id, events, and acting_user_id are required")
		return
	}

	var webhookCount int
	err := state.Pool.QueryRow(state.Context, "SELECT COUNT(1) FROM "+state.TableWebhooks+" WHERE id = $1 AND guild_id = $2", webhookId, req.GuildID).Scan(&webhookCount)
	if err != nil || webhookCount == 0 {
		writeError(w, http.StatusNotFound, "That webhook doesn't exist")
		return
	}

	if req.RepoID != nil && *req.RepoID != "" {
		var repoCount int
		err := state.Pool.QueryRow(state.Context, "SELECT COUNT(1) FROM "+state.TableRepos+" WHERE id = $1 AND webhook_id = $2", *req.RepoID, webhookId).Scan(&repoCount)
		if err != nil || repoCount == 0 {
			writeError(w, http.StatusBadRequest, "That repo doesn't exist on this webhook")
			return
		}
	}

	var modifierCount int
	err = state.Pool.QueryRow(state.Context, "SELECT COUNT(1) FROM "+state.TableEventModifiers+" WHERE webhook_id = $1", webhookId).Scan(&modifierCount)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error checking modifier count: "+err.Error())
		return
	}
	if modifierCount >= maxModifiersPerWebhook {
		writeError(w, http.StatusBadRequest, "You can only have 10 event modifiers per webhook!")
		return
	}

	id, err := randomAlphanumeric(32)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error generating modifier id")
		return
	}

	events := normalizeEvents(req.Events)

	_, err = state.Pool.Exec(
		state.Context,
		"INSERT INTO "+state.TableEventModifiers+" (id, webhook_id, events, repo_id, blacklisted, whitelisted, redirect_channel, guild_id, priority, created_by, last_updated_by) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)",
		id, webhookId, events, req.RepoID, req.Blacklisted, req.Whitelisted, req.RedirectChannel, req.GuildID, req.Priority, req.ActingUserID, req.ActingUserID,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error creating modifier: "+err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"id": id})
}

type updateModifierRequest struct {
	GuildID         string  `json:"guild_id"`
	Events          *string `json:"events"`
	Blacklisted     *bool   `json:"blacklisted"`
	Whitelisted     *bool   `json:"whitelisted"`
	Priority        *int    `json:"priority"`
	RepoID          *string `json:"repo_id"` // pass "" to clear
	RedirectChannel *string `json:"redirect_channel"`
	ActingUserID    string  `json:"acting_user_id"`
}

func ApiDashboardUpdateModifier(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req updateModifierRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if req.GuildID == "" || req.ActingUserID == "" {
		writeError(w, http.StatusBadRequest, "guild_id and acting_user_id are required")
		return
	}

	var webhookId string
	err := state.Pool.QueryRow(state.Context, "SELECT webhook_id FROM "+state.TableEventModifiers+" WHERE id = $1 AND guild_id = $2", id, req.GuildID).Scan(&webhookId)
	if err != nil {
		writeError(w, http.StatusNotFound, "That modifier doesn't exist")
		return
	}

	tx, err := state.Pool.Begin(state.Context)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error starting transaction: "+err.Error())
		return
	}
	defer tx.Rollback(state.Context)

	if req.Events != nil {
		events := normalizeEvents(*req.Events)
		if _, err := tx.Exec(state.Context, "UPDATE "+state.TableEventModifiers+" SET events = $1 WHERE id = $2", events, id); err != nil {
			writeError(w, http.StatusInternalServerError, "Error updating events: "+err.Error())
			return
		}
	}
	if req.Blacklisted != nil {
		if _, err := tx.Exec(state.Context, "UPDATE "+state.TableEventModifiers+" SET blacklisted = $1 WHERE id = $2", *req.Blacklisted, id); err != nil {
			writeError(w, http.StatusInternalServerError, "Error updating blacklisted: "+err.Error())
			return
		}
	}
	if req.Whitelisted != nil {
		if _, err := tx.Exec(state.Context, "UPDATE "+state.TableEventModifiers+" SET whitelisted = $1 WHERE id = $2", *req.Whitelisted, id); err != nil {
			writeError(w, http.StatusInternalServerError, "Error updating whitelisted: "+err.Error())
			return
		}
	}
	if req.Priority != nil {
		if _, err := tx.Exec(state.Context, "UPDATE "+state.TableEventModifiers+" SET priority = $1 WHERE id = $2", *req.Priority, id); err != nil {
			writeError(w, http.StatusInternalServerError, "Error updating priority: "+err.Error())
			return
		}
	}
	if req.RepoID != nil {
		var parsedRepoID *string
		if *req.RepoID != "" {
			var repoCount int
			if err := tx.QueryRow(state.Context, "SELECT COUNT(1) FROM "+state.TableRepos+" WHERE id = $1 AND webhook_id = $2", *req.RepoID, webhookId).Scan(&repoCount); err != nil || repoCount == 0 {
				writeError(w, http.StatusBadRequest, "That repo doesn't exist on this webhook")
				return
			}
			parsedRepoID = req.RepoID
		}
		if _, err := tx.Exec(state.Context, "UPDATE "+state.TableEventModifiers+" SET repo_id = $1 WHERE id = $2", parsedRepoID, id); err != nil {
			writeError(w, http.StatusInternalServerError, "Error updating repo scope: "+err.Error())
			return
		}
	}
	if req.RedirectChannel != nil {
		if _, err := tx.Exec(state.Context, "UPDATE "+state.TableEventModifiers+" SET redirect_channel = $1 WHERE id = $2", *req.RedirectChannel, id); err != nil {
			writeError(w, http.StatusInternalServerError, "Error updating redirect channel: "+err.Error())
			return
		}
	}

	if _, err := tx.Exec(state.Context, "UPDATE "+state.TableEventModifiers+" SET last_updated_by = $1 WHERE id = $2", req.ActingUserID, id); err != nil {
		writeError(w, http.StatusInternalServerError, "Error stamping last_updated_by: "+err.Error())
		return
	}

	if err := tx.Commit(state.Context); err != nil {
		writeError(w, http.StatusInternalServerError, "Error committing transaction: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type analyticsDay struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type analyticsEventType struct {
	EventType string `json:"event_type"`
	Count     int    `json:"count"`
}

func ApiDashboardAnalytics(w http.ResponseWriter, r *http.Request) {
	guildId := chi.URLParam(r, "guildId")

	days := 30
	if v := r.URL.Query().Get("days"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			days = parsed
		}
	}
	if days > 90 {
		days = 90
	}

	webhookId := r.URL.Query().Get("webhook_id")

	perDayRows, err := state.Pool.Query(
		state.Context,
		`SELECT date_trunc('day', m.occurred_at) AS day, COUNT(*) AS count
		FROM `+state.TableEventMetrics+` m
		JOIN `+state.TableWebhooks+` w ON w.id = m.webhook_id
		WHERE w.guild_id = $1
			AND m.occurred_at >= NOW() - ($2 || ' days')::interval
			AND ($3 = '' OR m.webhook_id = $3)
		GROUP BY day
		ORDER BY day ASC`,
		guildId, days, webhookId,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error aggregating analytics: "+err.Error())
		return
	}

	perDay := []analyticsDay{}
	for perDayRows.Next() {
		var day time.Time
		var count int
		if err := perDayRows.Scan(&day, &count); err != nil {
			continue
		}
		perDay = append(perDay, analyticsDay{Date: day.Format("2006-01-02"), Count: count})
	}
	perDayRows.Close()

	byTypeRows, err := state.Pool.Query(
		state.Context,
		`SELECT m.event_type, COUNT(*) AS count
		FROM `+state.TableEventMetrics+` m
		JOIN `+state.TableWebhooks+` w ON w.id = m.webhook_id
		WHERE w.guild_id = $1
			AND m.occurred_at >= NOW() - ($2 || ' days')::interval
			AND ($3 = '' OR m.webhook_id = $3)
		GROUP BY m.event_type
		ORDER BY count DESC`,
		guildId, days, webhookId,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error aggregating analytics by type: "+err.Error())
		return
	}

	byType := []analyticsEventType{}
	for byTypeRows.Next() {
		var eventType string
		var count int
		if err := byTypeRows.Scan(&eventType, &count); err != nil {
			continue
		}
		byType = append(byType, analyticsEventType{EventType: eventType, Count: count})
	}
	byTypeRows.Close()

	writeJSON(w, http.StatusOK, map[string]any{
		"per_day": perDay,
		"by_type": byType,
	})
}

func ApiDashboardDeleteModifier(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var req deleteRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if req.GuildID == "" {
		writeError(w, http.StatusBadRequest, "guild_id is required")
		return
	}

	if _, err := state.Pool.Exec(state.Context, "DELETE FROM "+state.TableEventModifiers+" WHERE id = $1 AND guild_id = $2", id, req.GuildID); err != nil {
		writeError(w, http.StatusInternalServerError, "Error deleting modifier: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

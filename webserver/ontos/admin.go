//  Copyright (C) 2026 NodeByte LTD

package ontos

import (
	"net/http"
	"strconv"
	"time"

	"github.com/OctoHubOSS/Octoflow/webserver/state"
	"github.com/go-chi/chi/v5"
)

func AdminAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userId := r.Header.Get("X-Acting-User-Id")

		if !state.IsAdmin(userId) {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte("Forbidden"))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func logAdminAction(adminUserId, action, target, detail string) {
	go func() {
		_, err := state.Pool.Exec(
			state.Context,
			"INSERT INTO "+state.TableAdminAuditLog+" (admin_user_id, action, target, detail) VALUES ($1, $2, $3, $4)",
			adminUserId, action, target, detail,
		)
		if err != nil {
			state.Logger.Sugar().Errorw("Could not write admin audit log", "error", err, "action", action)
		}
	}()
}

func ApiAdminWhoAmI(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"is_admin": true})
}

type adminStats struct {
	TotalGuilds        int    `json:"total_guilds"`
	BannedGuilds       int    `json:"banned_guilds"`
	TotalWebhooks      int    `json:"total_webhooks"`
	BrokenWebhooks     int    `json:"broken_webhooks"`
	TotalRepos         int    `json:"total_repos"`
	EventsLast24h      int    `json:"events_last_24h"`
	EventsLast30d      int    `json:"events_last_30d"`
	BotGuildCount      int    `json:"bot_guild_count"`
	BotMemberCount     int64  `json:"bot_member_count"`
	BotShardCount      int    `json:"bot_shard_count"`
	HeartbeatUpdatedAt string `json:"heartbeat_updated_at,omitempty"`
}

func ApiAdminStats(w http.ResponseWriter, r *http.Request) {
	var s adminStats

	if err := state.Pool.QueryRow(state.Context, "SELECT COUNT(*), COUNT(*) FILTER (WHERE banned) FROM "+state.TableGuilds).Scan(&s.TotalGuilds, &s.BannedGuilds); err != nil {
		writeError(w, http.StatusInternalServerError, "Error counting guilds: "+err.Error())
		return
	}

	if err := state.Pool.QueryRow(state.Context, "SELECT COUNT(*), COUNT(*) FILTER (WHERE broken) FROM "+state.TableWebhooks).Scan(&s.TotalWebhooks, &s.BrokenWebhooks); err != nil {
		writeError(w, http.StatusInternalServerError, "Error counting webhooks: "+err.Error())
		return
	}

	if err := state.Pool.QueryRow(state.Context, "SELECT COUNT(*) FROM "+state.TableRepos).Scan(&s.TotalRepos); err != nil {
		writeError(w, http.StatusInternalServerError, "Error counting repos: "+err.Error())
		return
	}

	if err := state.Pool.QueryRow(state.Context, "SELECT COUNT(*) FROM "+state.TableEventMetrics+" WHERE occurred_at >= NOW() - INTERVAL '24 hours'").Scan(&s.EventsLast24h); err != nil {
		writeError(w, http.StatusInternalServerError, "Error counting recent events: "+err.Error())
		return
	}

	if err := state.Pool.QueryRow(state.Context, "SELECT COUNT(*) FROM "+state.TableEventMetrics+" WHERE occurred_at >= NOW() - INTERVAL '30 days'").Scan(&s.EventsLast30d); err != nil {
		writeError(w, http.StatusInternalServerError, "Error counting monthly events: "+err.Error())
		return
	}

	var heartbeatAt time.Time
	err := state.Pool.QueryRow(
		state.Context,
		"SELECT guild_count, member_count, shard_count, updated_at FROM "+state.TableBotHeartbeat+" WHERE id = 1",
	).Scan(&s.BotGuildCount, &s.BotMemberCount, &s.BotShardCount, &heartbeatAt)
	if err == nil {
		s.HeartbeatUpdatedAt = heartbeatAt.Format(time.RFC3339)
	}

	writeJSON(w, http.StatusOK, s)
}

type adminGuild struct {
	ID           string `json:"id"`
	Name         string `json:"name,omitempty"`
	Icon         string `json:"icon,omitempty"`
	Banned       bool   `json:"banned"`
	WebhookCount int    `json:"webhook_count"`
	RepoCount    int    `json:"repo_count"`
}

func ApiAdminGuilds(w http.ResponseWriter, r *http.Request) {
	rows, err := state.Pool.Query(
		state.Context,
		`SELECT g.id, g.banned,
			(SELECT COUNT(*) FROM `+state.TableWebhooks+` w WHERE w.guild_id = g.id) AS webhook_count,
			(SELECT COUNT(*) FROM `+state.TableRepos+` rp WHERE rp.guild_id = g.id) AS repo_count
		FROM `+state.TableGuilds+` g
		ORDER BY webhook_count DESC`,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error fetching guilds: "+err.Error())
		return
	}
	defer rows.Close()

	guilds := []adminGuild{}
	for rows.Next() {
		var g adminGuild
		if err := rows.Scan(&g.ID, &g.Banned, &g.WebhookCount, &g.RepoCount); err != nil {
			writeError(w, http.StatusInternalServerError, "Error scanning guild: "+err.Error())
			return
		}
		guilds = append(guilds, g)
	}

	for i := range guilds {
		gd, err := state.Discord.Guild(guilds[i].ID)
		if err == nil && gd != nil {
			guilds[i].Name = gd.Name
			guilds[i].Icon = gd.Icon
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"guilds": guilds})
}

type banGuildRequest struct {
	Banned       bool   `json:"banned"`
	ActingUserID string `json:"acting_user_id"`
}

func ApiAdminBanGuild(w http.ResponseWriter, r *http.Request) {
	guildId := chi.URLParam(r, "guildId")

	var req banGuildRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if req.ActingUserID == "" {
		writeError(w, http.StatusBadRequest, "acting_user_id is required")
		return
	}

	result, err := state.Pool.Exec(state.Context, "UPDATE "+state.TableGuilds+" SET banned = $1 WHERE id = $2", req.Banned, guildId)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error updating guild: "+err.Error())
		return
	}
	if result.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "Guild not found")
		return
	}

	action := "unban_guild"
	if req.Banned {
		action = "ban_guild"
	}
	logAdminAction(req.ActingUserID, action, guildId, "")

	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

type adminLogEntry struct {
	LogID     string   `json:"log_id"`
	GuildID   string   `json:"guild_id"`
	WebhookID string   `json:"webhook_id"`
	Entries   []string `json:"entries"`
}

func ApiAdminLogs(w http.ResponseWriter, r *http.Request) {
	guildId := r.URL.Query().Get("guild_id")
	webhookId := r.URL.Query().Get("webhook_id")

	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}
	offset := 0
	if v := r.URL.Query().Get("offset"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	rows, err := state.Pool.Query(
		state.Context,
		`SELECT log_id, guild_id, webhook_id, entries FROM `+state.TableWebhookLogs+`
		WHERE ($1 = '' OR guild_id = $1) AND ($2 = '' OR webhook_id = $2)
		ORDER BY log_id DESC
		LIMIT $3 OFFSET $4`,
		guildId, webhookId, limit, offset,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error fetching logs: "+err.Error())
		return
	}
	defer rows.Close()

	logs := []adminLogEntry{}
	for rows.Next() {
		var l adminLogEntry
		if err := rows.Scan(&l.LogID, &l.GuildID, &l.WebhookID, &l.Entries); err != nil {
			writeError(w, http.StatusInternalServerError, "Error scanning log: "+err.Error())
			return
		}
		logs = append(logs, l)
	}

	writeJSON(w, http.StatusOK, map[string]any{"logs": logs})
}

type adminAuditRow struct {
	ID          int64  `json:"id"`
	AdminUserID string `json:"admin_user_id"`
	Action      string `json:"action"`
	Target      string `json:"target,omitempty"`
	Detail      string `json:"detail,omitempty"`
	CreatedAt   string `json:"created_at"`
}

func ApiAdminAuditLog(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if v := r.URL.Query().Get("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}

	rows, err := state.Pool.Query(
		state.Context,
		"SELECT id, admin_user_id, action, COALESCE(target, ''), COALESCE(detail, ''), created_at FROM "+state.TableAdminAuditLog+" ORDER BY id DESC LIMIT $1",
		limit,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error fetching audit log: "+err.Error())
		return
	}
	defer rows.Close()

	entries := []adminAuditRow{}
	for rows.Next() {
		var e adminAuditRow
		var createdAt time.Time
		if err := rows.Scan(&e.ID, &e.AdminUserID, &e.Action, &e.Target, &e.Detail, &createdAt); err != nil {
			writeError(w, http.StatusInternalServerError, "Error scanning audit row: "+err.Error())
			return
		}
		e.CreatedAt = createdAt.Format(time.RFC3339)
		entries = append(entries, e)
	}

	writeJSON(w, http.StatusOK, map[string]any{"entries": entries})
}

//  Copyright (C) 2026 NodeByte LTD

package ontos

import (
	"net/http"

	"github.com/OctoHubOSS/Octoflow/webserver/state"
)

// publicStats is deliberately a pure-aggregate view - no guild IDs, names,
// or anything else that identifies a specific server, unlike the admin
// panel's stats endpoint. Safe to expose with no auth at all, same trust
// level as /api/health and /api/counts.
type publicStats struct {
	TotalWebhooks int   `json:"total_webhooks"`
	TotalRepos    int   `json:"total_repos"`
	EventsLast24h int   `json:"events_last_24h"`
	EventsLast7d  int   `json:"events_last_7d"`
	EventsLast30d int   `json:"events_last_30d"`
	EventsAllTime int   `json:"events_all_time"`
	GuildCount    int   `json:"guild_count"`
	MemberCount   int64 `json:"member_count"`
	ShardCount    int   `json:"shard_count"`
}

func ApiPublicStats(w http.ResponseWriter, r *http.Request) {
	var s publicStats

	if err := state.Pool.QueryRow(state.Context, "SELECT COUNT(*) FROM "+state.TableWebhooks).Scan(&s.TotalWebhooks); err != nil {
		writeError(w, http.StatusInternalServerError, "Error counting webhooks: "+err.Error())
		return
	}

	if err := state.Pool.QueryRow(state.Context, "SELECT COUNT(*) FROM "+state.TableRepos).Scan(&s.TotalRepos); err != nil {
		writeError(w, http.StatusInternalServerError, "Error counting repos: "+err.Error())
		return
	}

	if err := state.Pool.QueryRow(state.Context, "SELECT COUNT(*) FROM "+state.TableEventMetrics+" WHERE occurred_at >= NOW() - INTERVAL '24 hours'").Scan(&s.EventsLast24h); err != nil {
		writeError(w, http.StatusInternalServerError, "Error counting 24h events: "+err.Error())
		return
	}

	if err := state.Pool.QueryRow(state.Context, "SELECT COUNT(*) FROM "+state.TableEventMetrics+" WHERE occurred_at >= NOW() - INTERVAL '7 days'").Scan(&s.EventsLast7d); err != nil {
		writeError(w, http.StatusInternalServerError, "Error counting 7d events: "+err.Error())
		return
	}

	if err := state.Pool.QueryRow(state.Context, "SELECT COUNT(*) FROM "+state.TableEventMetrics+" WHERE occurred_at >= NOW() - INTERVAL '30 days'").Scan(&s.EventsLast30d); err != nil {
		writeError(w, http.StatusInternalServerError, "Error counting 30d events: "+err.Error())
		return
	}

	if err := state.Pool.QueryRow(state.Context, "SELECT COUNT(*) FROM "+state.TableEventMetrics).Scan(&s.EventsAllTime); err != nil {
		writeError(w, http.StatusInternalServerError, "Error counting all-time events: "+err.Error())
		return
	}

	// Best-effort - a missing heartbeat row just leaves these zeroed rather
	// than failing the whole response, same as /api/health's own handling.
	_ = state.Pool.QueryRow(
		state.Context,
		"SELECT guild_count, member_count, shard_count FROM "+state.TableBotHeartbeat+" WHERE id = 1",
	).Scan(&s.GuildCount, &s.MemberCount, &s.ShardCount)

	writeJSON(w, http.StatusOK, s)
}

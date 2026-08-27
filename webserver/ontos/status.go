//  Copyright (C) 2026 NodeByte LTD

package ontos

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/OctoHubOSS/Octoflow/webserver/state"
)

const heartbeatStaleAfter = 3 * time.Minute

type healthResponse struct {
	Database    bool      `json:"database"`
	Discord     bool      `json:"discord"`
	GuildCount  int       `json:"guild_count"`
	MemberCount int64     `json:"member_count"`
	ShardCount  int       `json:"shard_count"`
	CheckedAt   time.Time `json:"checked_at"`
}

func ApiHealth(w http.ResponseWriter, r *http.Request) {
	resp := healthResponse{CheckedAt: time.Now().UTC()}

	var updatedAt time.Time
	err := state.Pool.QueryRow(
		state.Context,
		"SELECT guild_count, member_count, shard_count, updated_at FROM "+state.TableBotHeartbeat+" WHERE id = 1",
	).Scan(&resp.GuildCount, &resp.MemberCount, &resp.ShardCount, &updatedAt)

	resp.Database = err == nil
	resp.Discord = err == nil && time.Since(updatedAt) < heartbeatStaleAfter

	w.Header().Set("Content-Type", "application/json")
	if !resp.Database {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	json.NewEncoder(w).Encode(resp)
}

func ApiStats(w http.ResponseWriter, r *http.Request) {
	var guildCount, shardCount int
	var memberCount int64

	err := state.Pool.QueryRow(
		state.Context,
		"SELECT guild_count, member_count, shard_count FROM "+state.TableBotHeartbeat+" WHERE id = 1",
	).Scan(&guildCount, &memberCount, &shardCount)

	if err != nil {
		w.Write([]byte("0,0,0"))
		return
	}

	w.Write([]byte(strconv.Itoa(guildCount) + "," + strconv.FormatInt(memberCount, 10) + "," + strconv.Itoa(shardCount)))
}

type dayUptime struct {
	Date          string  `json:"date"`
	UptimePercent float64 `json:"uptime_percent"`
	AvgLatencyMs  float64 `json:"avg_latency_ms"`
	Checks        int     `json:"checks"`
}

func ApiStatusHistory(w http.ResponseWriter, r *http.Request) {
	days := 30

	if v := r.URL.Query().Get("days"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			days = parsed
		}
	}

	if days > 90 {
		days = 90
	}

	rows, err := state.Pool.Query(
		state.Context,
		`SELECT
			date_trunc('day', checked_at) AS day,
			100.0 * AVG(CASE WHEN database_up AND discord_up THEN 1 ELSE 0 END) AS uptime_percent,
			AVG(db_latency_ms) AS avg_latency_ms,
			COUNT(*) AS checks
		FROM `+state.TableStatusSnapshots+`
		WHERE checked_at >= NOW() - ($1 || ' days')::interval
		GROUP BY day
		ORDER BY day ASC`,
		days,
	)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error fetching status history: " + err.Error()))
		return
	}
	defer rows.Close()

	history := []dayUptime{}

	for rows.Next() {
		var day time.Time
		var d dayUptime

		if err := rows.Scan(&day, &d.UptimePercent, &d.AvgLatencyMs, &d.Checks); err != nil {
			continue
		}

		d.Date = day.Format("2006-01-02")
		history = append(history, d)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(history)
}

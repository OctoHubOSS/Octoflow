//  Copyright (C) 2026 NodeByte LTD

package ontos

import (
	"time"

	"github.com/OctoHubOSS/Octoflow/webserver/state"
	"go.uber.org/zap"
)

const snapshotInterval = 5 * time.Minute

func StartStatusSnapshotter() {
	ticker := time.NewTicker(snapshotInterval)
	defer ticker.Stop()

	takeSnapshot()

	for range ticker.C {
		takeSnapshot()
	}
}

func takeSnapshot() {
	start := time.Now()

	var updatedAt time.Time
	err := state.Pool.QueryRow(
		state.Context,
		"SELECT updated_at FROM "+state.TableBotHeartbeat+" WHERE id = 1",
	).Scan(&updatedAt)

	latency := time.Since(start)

	databaseUp := err == nil
	discordUp := err == nil && time.Since(updatedAt) < heartbeatStaleAfter

	_, insertErr := state.Pool.Exec(
		state.Context,
		"INSERT INTO "+state.TableStatusSnapshots+" (database_up, discord_up, db_latency_ms) VALUES ($1, $2, $3)",
		databaseUp, discordUp, latency.Milliseconds(),
	)

	if insertErr != nil {
		state.Logger.Error("Failed to write status snapshot", zap.Error(insertErr))
	}
}

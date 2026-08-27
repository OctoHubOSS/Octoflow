//  Copyright (C) 2026 NodeByte LTD

package pneuma

import (
	"fmt"

	"github.com/OctoHubOSS/Octoflow/webserver/logos/events"
	"github.com/OctoHubOSS/Octoflow/webserver/state"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

func resolveThreadKey(repoId, header string, bodyBytes []byte) (events.ThreadKey, bool) {
	var useThreads bool
	if err := state.Pool.QueryRow(state.Context, "SELECT use_threads FROM "+state.TableRepos+" WHERE id = $1", repoId).Scan(&useThreads); err != nil || !useThreads {
		return events.ThreadKey{}, false
	}

	return events.ExtractThreadKey(header, bodyBytes)
}

func lookupIssueThread(repoId string, key events.ThreadKey) (string, bool) {
	var threadId string
	err := state.Pool.QueryRow(
		state.Context,
		"SELECT thread_id FROM "+state.TableIssueThreads+" WHERE repo_id = $1 AND issue_number = $2 AND kind = $3",
		repoId, key.Number, key.Kind,
	).Scan(&threadId)

	if err != nil || threadId == "" {
		return "", false
	}

	return threadId, true
}

func createIssueThread(logId, webhookId, guildId, repoId, channelId, messageId string, key events.ThreadKey) {
	name := fmt.Sprintf("#%d %s", key.Number, key.Title)
	if len(name) > 100 {
		name = name[:100]
	}

	thread, err := state.Discord.MessageThreadStartComplex(channelId, messageId, &discordgo.ThreadStart{
		Name:                name,
		AutoArchiveDuration: 1440,
	})

	if err != nil {
		updateLogEntries(logId, webhookId, guildId, "Could not create thread for #"+fmt.Sprint(key.Number)+": "+err.Error())
		state.Logger.Warn("Could not create issue thread", zap.Error(err), zap.String("repoID", repoId), zap.Int("number", key.Number))
		return
	}

	_, err = state.Pool.Exec(
		state.Context,
		"INSERT INTO "+state.TableIssueThreads+" (repo_id, issue_number, kind, thread_id) VALUES ($1, $2, $3, $4) ON CONFLICT (repo_id, issue_number, kind) DO NOTHING",
		repoId, key.Number, key.Kind, thread.ID,
	)

	if err != nil {
		updateLogEntries(logId, webhookId, guildId, "Could not save thread mapping: "+err.Error())
		state.Logger.Warn("Could not save issue thread mapping", zap.Error(err), zap.String("repoID", repoId), zap.Int("number", key.Number))
	}
}

package pneuma

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/OctoHubOSS/Octoflow/webserver/logos/eventmodifiers"
	"github.com/OctoHubOSS/Octoflow/webserver/logos/events"
	"github.com/OctoHubOSS/Octoflow/webserver/state"
	"go.uber.org/zap"
)

const batchWindow = 20 * time.Second

type pendingBatch struct {
	webhookId  string
	repoId     string
	guildId    string
	firstLogId string
	bodies     [][]byte
}

var (
	batchMu sync.Mutex
	batches = map[string]*pendingBatch{}
)

func enqueueForBatching(webhookId, repoId, guildId, logId string, bodyBytes []byte) {
	key := webhookId + ":" + repoId

	batchMu.Lock()
	b, exists := batches[key]
	if !exists {
		b = &pendingBatch{webhookId: webhookId, repoId: repoId, guildId: guildId, firstLogId: logId}
		batches[key] = b
		time.AfterFunc(batchWindow, func() { flushBatch(key) })
	}
	b.bodies = append(b.bodies, bodyBytes)
	batchMu.Unlock()
}

func flushBatch(key string) {
	batchMu.Lock()
	b, exists := batches[key]
	if exists {
		delete(batches, key)
	}
	batchMu.Unlock()

	if !exists || len(b.bodies) == 0 {
		return
	}

	l := state.MapMutex.Lock(b.webhookId)
	defer l.Unlock()

	updateLogEntries(b.firstLogId, b.webhookId, b.guildId, "Flushing batch of push events", len(b.bodies))

	modres, err := eventmodifiers.CheckEventAllowed(b.webhookId, b.repoId, "push")

	if err != nil || modres == nil {
		state.Logger.Error("Batch flush: error checking event modifiers", zap.Error(err), zap.String("webhookID", b.webhookId))
		return
	}

	if modres.ACLFail != "" {
		updateLogEntries(b.firstLogId, b.webhookId, b.guildId, "Batch ACL Fail: acl="+modres.ACLFail)
		return
	}

	var channelIds []string

	if modres.ChannelOverride != "" {
		channelIds = []string{modres.ChannelOverride}
	} else {
		var wrapper events.RepoWrapper
		for _, body := range b.bodies {
			if err := json.Unmarshal(body, &wrapper); err == nil && wrapper.Repo.FullName != "" {
				break
			}
		}

		rows, err := state.Pool.Query(state.Context, "SELECT channel_id FROM "+state.TableRepos+" WHERE repo_name = $1 AND webhook_id = $2", strings.ToLower(wrapper.Repo.FullName), b.webhookId)
		if err != nil {
			state.Logger.Error("Batch flush: channel fetch error", zap.Error(err), zap.String("webhookID", b.webhookId))
			return
		}
		defer rows.Close()

		for rows.Next() {
			var channelId string
			if err := rows.Scan(&channelId); err == nil {
				channelIds = append(channelIds, channelId)
			}
		}
	}

	if len(channelIds) == 0 {
		return
	}

	messageSend, err := events.BatchedPushFn(b.bodies)
	if err != nil {
		state.Logger.Error("Batch flush: error building combined embed", zap.Error(err), zap.String("webhookID", b.webhookId))
		return
	}

	for i, embed := range messageSend.Embeds {
		messageSend.Embeds[i] = applyEmbedLimits(embed)
	}

	for _, channelId := range channelIds {
		if _, err := state.Discord.ChannelMessageSendComplex(channelId, messageSend); err != nil {
			updateLogEntries(b.firstLogId, b.webhookId, b.guildId, "Batch send error: channelId="+channelId+" err="+err.Error())
		}
	}

	for range b.bodies {
		if _, err := state.Pool.Exec(state.Context,
			"INSERT INTO "+state.TableEventMetrics+" (webhook_id, repo_id, event_type) VALUES ($1, $2, 'push')",
			b.webhookId, b.repoId,
		); err != nil {
			state.Logger.Warn("Could not record batched event metric", zap.Error(err), zap.String("webhookID", b.webhookId))
		}
	}
}

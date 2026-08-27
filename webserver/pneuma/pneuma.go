//  Copyright (C) 2026 NodeByte LTD

package pneuma

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/OctoHubOSS/Octoflow/webserver/logos/eventmodifiers"
	"github.com/OctoHubOSS/Octoflow/webserver/logos/events"
	"github.com/OctoHubOSS/Octoflow/webserver/state"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

const (
	EMBED_TITLE_LIMIT       = 256
	EMBED_DESCRIPTION_LIMIT = 4096
	EMBED_FIELDS_MAX_COUNT  = 25
	EMBED_FIELD_NAME_LIMIT  = 256
	EMBED_FIELD_VALUE_LIMIT = 1024
	EMBED_FOOTER_TEXT_LIMIT = 2048
	EMBED_AUTHOR_NAME_LIMIT = 256
	EMBED_TOTAL_LIMIT       = 6000
)

func updateLogEntries(logId, webhookId, guildId string, entries ...any) error {
	var count int

	err := state.Pool.QueryRow(state.Context, "SELECT COUNT(*) FROM "+state.TableWebhookLogs+" WHERE log_id = $1 AND webhook_id = $2 AND guild_id = $3", logId, webhookId, guildId).Scan(&count)

	if err != nil {
		return err
	}

	entry := fmt.Sprintln(entries...)

	if count == 0 {
		_, err = state.Pool.Exec(state.Context, "INSERT INTO "+state.TableWebhookLogs+" (log_id, webhook_id, guild_id, entries) VALUES ($1, $2, $3, $4)", logId, webhookId, guildId, []string{entry})
		return err
	}

	_, err = state.Pool.Exec(state.Context, "UPDATE "+state.TableWebhookLogs+" SET entries = array_append(entries, $1) WHERE log_id = $2 AND webhook_id = $3 AND guild_id = $4", entry, logId, webhookId, guildId)
	return err
}

func applyEmbedLimits(e *discordgo.MessageEmbed) *discordgo.MessageEmbed {
	totalChars := 0

	_getCharLimit := func(totalChars, limit, maxChars int) int {
		if maxChars <= totalChars {
			return 0
		}

		return min(limit, maxChars-totalChars)
	}

	_sliceChars := func(s string, totalChars *int, limit, maxChars int) string {
		charLimit := _getCharLimit(*totalChars, limit, maxChars)

		if charLimit == 0 {
			return ""
		}

		if len(s) <= charLimit {
			*totalChars += len(s)
			return s
		} else {
			*totalChars += charLimit
			return s[:charLimit]
		}
	}

	if e.Title != "" {
		e.Title = _sliceChars(e.Title, &totalChars, EMBED_TITLE_LIMIT, EMBED_TOTAL_LIMIT)
	}

	if e.Description != "" {
		e.Description = _sliceChars(e.Description, &totalChars, EMBED_DESCRIPTION_LIMIT, EMBED_TOTAL_LIMIT)
	}

	if len(e.Fields) > EMBED_FIELDS_MAX_COUNT {
		e.Fields = e.Fields[:EMBED_FIELDS_MAX_COUNT]
	}

	for i, f := range e.Fields {
		e.Fields[i].Name = _sliceChars(f.Name, &totalChars, EMBED_FIELD_NAME_LIMIT, EMBED_TOTAL_LIMIT)
		e.Fields[i].Value = _sliceChars(f.Value, &totalChars, EMBED_FIELD_VALUE_LIMIT, EMBED_TOTAL_LIMIT)
	}

	if e.Footer != nil {
		e.Footer.Text = _sliceChars(e.Footer.Text, &totalChars, EMBED_FOOTER_TEXT_LIMIT, EMBED_TOTAL_LIMIT)
	}

	if e.Author != nil {
		e.Author.Name = _sliceChars(e.Author.Name, &totalChars, EMBED_AUTHOR_NAME_LIMIT, EMBED_TOTAL_LIMIT)
	}

	return e
}

func HandleEvents(
	bodyBytes []byte,
	rw *events.RepoWrapper,
	repoId string,
	logId string,
	header string,
	webhookId string,
	guildId string,
) {
	// Push events on a webhook with batch_events on are diverted to the
	// batcher instead of being dispatched immediately - see batcher.go. Every
	// other event type (and push on webhooks without it enabled) is
	// unaffected.
	if header == "push" {
		var batchEvents bool
		err := state.Pool.QueryRow(state.Context, "SELECT batch_events FROM "+state.TableWebhooks+" WHERE id = $1", webhookId).Scan(&batchEvents)
		if err == nil && batchEvents {
			enqueueForBatching(webhookId, repoId, guildId, logId, bodyBytes)
			return
		}
	}

	dispatchEvent(bodyBytes, rw, repoId, logId, header, webhookId, guildId)
}

// dispatchEvent does the actual work: resolve channels, render the embed,
// record analytics, and send - either called directly by HandleEvents, or
// by the batcher once a batch window flushes.
func dispatchEvent(
	bodyBytes []byte,
	rw *events.RepoWrapper,
	repoId string,
	logId string,
	header string,
	webhookId string,
	guildId string,
) {
	l := state.MapMutex.Lock(webhookId)
	defer l.Unlock()

	updateLogEntries(logId, webhookId, guildId, "Processing event: "+header, "repoName="+rw.Repo.FullName, "webhookID="+webhookId, "event="+header, "logId="+logId)

	modres, err := eventmodifiers.CheckEventAllowed(webhookId, repoId, header)

	if err != nil {
		updateLogEntries(logId, webhookId, guildId, "Error checking event modifiers: "+err.Error())
		state.Logger.Error("Error checking event modifiers", zap.Error(err), zap.String("repoName", rw.Repo.FullName), zap.String("webhookID", webhookId), zap.String("logId", logId))
		return
	}

	if modres == nil {
		updateLogEntries(logId, webhookId, guildId, "Internal Error: modres is nil")
		state.Logger.Error("Internal Error: modres is nil")
		return
	}

	if modres.ACLFail != "" {
		updateLogEntries(logId, webhookId, guildId, "ACL Fail: acl="+modres.ACLFail)
		state.Logger.Warn("ACL Fail", zap.String("repoName", rw.Repo.FullName), zap.String("webhookID", webhookId), zap.String("event", header), zap.String("reason", modres.ACLFail), zap.String("logId", logId))
		return
	}

	// Fire-and-forget: the event has cleared ACL checks, so it's going to be
	// delivered somewhere. Record it for dashboard analytics regardless of
	// per-channel send outcome below.
	go func() {
		if _, err := state.Pool.Exec(state.Context,
			"INSERT INTO "+state.TableEventMetrics+" (webhook_id, repo_id, event_type) VALUES ($1, $2, $3)",
			webhookId, repoId, header,
		); err != nil {
			state.Logger.Warn("Could not record event metric", zap.Error(err), zap.String("webhookID", webhookId), zap.String("event", header))
		}
	}()

	var channelIds []string

	if modres.ChannelOverride != "" {
		channelIds = []string{modres.ChannelOverride}
	} else {
		rows, err := state.Pool.Query(state.Context, "SELECT channel_id FROM "+state.TableRepos+" WHERE repo_name = $1 AND webhook_id = $2", strings.ToLower(rw.Repo.FullName), webhookId)

		if err != nil {
			updateLogEntries(logId, "Channel id fetch error: acl="+modres.ACLFail, "error="+err.Error())
			state.Logger.Error("Channel id fetch error", zap.Error(err), zap.String("repoName", rw.Repo.FullName), zap.String("webhookID", webhookId), zap.String("logId", logId))
			return
		}

		defer rows.Close()

		for rows.Next() {
			var channelId string

			err = rows.Scan(&channelId)

			if err != nil {
				updateLogEntries(logId, "Channel id scan error: acl="+modres.ACLFail, "error="+err.Error())
				state.Logger.Error("Channel id scan error", zap.Error(err), zap.String("repoName", rw.Repo.FullName), zap.String("webhookID", webhookId), zap.String("logId", logId))
				continue
			}

			channelIds = append(channelIds, channelId)
		}
	}

	if len(channelIds) == 0 {
		return
	}

	evtFn, ok := events.SupportedEvents[header]

	var messageSend *discordgo.MessageSend

	if !ok {
		updateLogEntries(logId, webhookId, guildId, "WARNING: This event cannot be personalized, will try propogating to configured webhooks (if supported)?")

		var fields map[string]any

		if err := json.Unmarshal(bodyBytes, &fields); err != nil {
			updateLogEntries(logId, webhookId, guildId, "Error unmarshalling event: "+err.Error())
			state.Logger.Error("Error unmarshalling event", zap.Error(err), zap.String("repoName", rw.Repo.FullName), zap.String("webhookID", webhookId), zap.String("logId", logId))
			return
		}

		var embed = discordgo.MessageEmbed{
			Title:  cases.Title(language.English).String(strings.ReplaceAll(header, "_", " ")),
			Fields: []*discordgo.MessageEmbedField{},
		}

		for k, v := range fields {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:  cases.Title(language.English).String(strings.ReplaceAll(k, "_", " ")),
				Value: cases.Title(language.English).String(strings.ReplaceAll(fmt.Sprintf("%v", v), "_", " ")),
			})
		}

		messageSend = &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{&embed},
		}
	} else {
		updateLogEntries(logId, webhookId, guildId, "SUCCESS: This event can be personalized")
		messageSend, err = evtFn(bodyBytes)

		if err != nil {
			updateLogEntries(logId, webhookId, guildId, "Error processing event:", err.Error())
			state.Logger.Error("Error processing event", zap.Error(err), zap.String("repoName", rw.Repo.FullName), zap.String("webhookID", webhookId), zap.String("event", header), zap.String("logId", logId))
			return
		}
	}

	for i, embed := range messageSend.Embeds {
		messageSend.Embeds[i] = applyEmbedLimits(embed)
	}

	// Thread-per-PR/issue mode only applies to a repo's own normal channel
	// routing, not an event-modifier redirect - a redirect target isn't "the
	// repo's channel" in the sense threading assumes.
	var threadKey events.ThreadKey
	var hasThreadKey bool
	if modres.ChannelOverride == "" {
		threadKey, hasThreadKey = resolveThreadKey(repoId, header, bodyBytes)
	}

	for _, channelId := range channelIds {
		target := channelId
		createThreadAfterSend := false

		if hasThreadKey {
			if existing, ok := lookupIssueThread(repoId, threadKey); ok {
				target = existing
			} else if threadKey.IsThreadOpeningEvent(header) {
				createThreadAfterSend = true
			}
		}

		updateLogEntries(logId, webhookId, guildId, "Sending event to channel: channelId="+target)
		sentMsg, err := state.Discord.ChannelMessageSendComplex(target, messageSend)

		if err != nil {
			state.Discord.ChannelMessageSendComplex(channelId, &discordgo.MessageSend{
				Content: "Could not send event " + header + " to channel: <#" + channelId + ">:" + err.Error(),
			})

			updateLogEntries(logId, "Could not send event "+header+" to channel: channelId="+channelId, "err="+err.Error())
			continue
		}

		if createThreadAfterSend {
			createIssueThread(logId, webhookId, guildId, repoId, target, sentMsg.ID, threadKey)
		}
	}
}

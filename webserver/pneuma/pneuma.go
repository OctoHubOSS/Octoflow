// Pneuma (Xenoblade Chronicles 2), the core component that actually handles events
package pneuma

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/git-logs/client/webserver/logos/eventmodifiers"
	"github.com/git-logs/client/webserver/logos/events"
	"github.com/git-logs/client/webserver/state"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

const (
	// EMBED_TITLE_LIMIT is the maximum length of an embed title
	EMBED_TITLE_LIMIT = 256
	// EMBED_DESCRIPTION_LIMIT is the maximum length of an embed description
	EMBED_DESCRIPTION_LIMIT = 4096
	// EMBED_FIELDS_MAX_COUNT is the maximum number of fields in an embed
	EMBED_FIELDS_MAX_COUNT = 25
	// EMBED_FIELD_NAME_LIMIT is the maximum length of an embed field name
	EMBED_FIELD_NAME_LIMIT = 256
	// EMBED_FIELD_VALUE_LIMIT is the maximum length of an embed field value
	EMBED_FIELD_VALUE_LIMIT = 1024
	// EMBED_FOOTER_TEXT_LIMIT is the maximum length of an embed footer text
	EMBED_FOOTER_TEXT_LIMIT = 2048
	// EMBED_AUTHOR_NAME_LIMIT is the maximum length of an embed author name
	EMBED_AUTHOR_NAME_LIMIT = 256
	// EMBED_TOTAL_LIMIT is the maximum length of an embed
	EMBED_TOTAL_LIMIT = 6000
)

// NormalizeGitHubEventHeader trims BOM/whitespace and lowercases GitHub's X-GitHub-Event value.
func NormalizeGitHubEventHeader(h string) string {
	h = strings.TrimPrefix(strings.TrimSpace(h), "\ufeff")
	return strings.ToLower(h)
}

func updateLogEntries(logId, webhookId, guildId string, entries ...any) error {
	// Check for log_id in database
	var count int

	err := state.Pool.QueryRow(state.Context, "SELECT COUNT(*) FROM "+state.TableWebhookLogs+" WHERE log_id = $1 AND webhook_id = $2 AND guild_id = $3", logId, webhookId, guildId).Scan(&count)

	if err != nil {
		return err
	}

	entry := fmt.Sprintln(entries...)

	if count == 0 {
		// Insert new log_id
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

		// If limit is 6000 and max_chars - total_chars is 1000, return 1000 etc.
		return min(limit, maxChars-totalChars)
	}

	_sliceChars := func(s string, totalChars *int, limit, maxChars int) string {
		charLimit := _getCharLimit(*totalChars, limit, maxChars)

		if charLimit == 0 {
			return ""
		}

		// Avoid panic as go doesn't handle slices out of bounds
		if len(s) <= charLimit {
			*totalChars += len(s)
			return s
		} else {
			*totalChars += charLimit
			return s[:charLimit]
		}
	}

	if e.Title != "" {
		// Slice title to EMBED_TITLE_LIMIT
		e.Title = _sliceChars(e.Title, &totalChars, EMBED_TITLE_LIMIT, EMBED_TOTAL_LIMIT)
	}

	if e.Description != "" {
		// Slice description to EMBED_DESCRIPTION_LIMIT
		e.Description = _sliceChars(e.Description, &totalChars, EMBED_DESCRIPTION_LIMIT, EMBED_TOTAL_LIMIT)
	}

	// Slice out fields if there are too many
	if len(e.Fields) > EMBED_FIELDS_MAX_COUNT {
		e.Fields = e.Fields[:EMBED_FIELDS_MAX_COUNT]
	}

	for i, f := range e.Fields {
		// Slice field name to EMBED_FIELD_NAME_LIMIT
		e.Fields[i].Name = _sliceChars(f.Name, &totalChars, EMBED_FIELD_NAME_LIMIT, EMBED_TOTAL_LIMIT)

		// Slice field value to EMBED_FIELD_VALUE_LIMIT
		e.Fields[i].Value = _sliceChars(f.Value, &totalChars, EMBED_FIELD_VALUE_LIMIT, EMBED_TOTAL_LIMIT)
	}

	if e.Footer != nil {
		// Slice footer text to EMBED_FOOTER_TEXT_LIMIT
		e.Footer.Text = _sliceChars(e.Footer.Text, &totalChars, EMBED_FOOTER_TEXT_LIMIT, EMBED_TOTAL_LIMIT)
	}

	if e.Author != nil {
		// Slice author name to EMBED_AUTHOR_NAME_LIMIT
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
	provider string,
) {
	// Ensure one at a time
	l := state.MapMutex.Lock(webhookId)
	defer l.Unlock()

	header = NormalizeGitHubEventHeader(header)

	updateLogEntries(logId, webhookId, guildId, "Processing event: "+header, "provider="+provider, "repoName="+rw.Repo.FullName, "webhookID="+webhookId, "event="+header, "logId="+logId)

	// Check event modifiers
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

	var channelIds []string

	// Channel override comes from the event modifier, in the case of an event modifier, we only send
	// to the channel specified in the event modifier, not to all channels set
	if modres.ChannelOverride != "" {
		channelIds = []string{modres.ChannelOverride}
	} else {
		// Get channel ID from database
		rows, err := state.Pool.Query(state.Context, "SELECT channel_id FROM "+state.TableRepos+" WHERE repo_name = $1 AND webhook_id = $2", strings.ToLower(rw.Repo.FullName), webhookId)

		if err != nil {
			updateLogEntries(logId, webhookId, guildId, "Channel id fetch error: acl="+modres.ACLFail, "error="+err.Error())
			state.Logger.Error("Channel id fetch error", zap.Error(err), zap.String("repoName", rw.Repo.FullName), zap.String("webhookID", webhookId), zap.String("logId", logId))
			return
		}

		defer rows.Close()

		for rows.Next() {
			var channelId string

			err = rows.Scan(&channelId)

			if err != nil {
				updateLogEntries(logId, webhookId, guildId, "Channel id scan error: acl="+modres.ACLFail, "error="+err.Error())
				state.Logger.Error("Channel id scan error", zap.Error(err), zap.String("repoName", rw.Repo.FullName), zap.String("webhookID", webhookId), zap.String("logId", logId))
				continue
			}

			channelIds = append(channelIds, channelId)
		}
	}

	// Early return, don't waste resources if there are no channels to send to
	if len(channelIds) == 0 {
		return
	}

	// Try provider-specific supported events first, then fall back
	var evtFn func([]byte) (*discordgo.MessageSend, error)
	var ok bool

	if provider == "gitlab" {
		evtFn, ok = events.GitLabSupportedEvents[header]
	} else {
		evtFn, ok = events.SupportedEvents[header]
		if !ok && events.LooksLikeRepositoryVulnerabilityAlert(bodyBytes) {
			evtFn = events.RepositoryVulnerabilityAlert
			ok = true
		}
	}

	var messageSend *discordgo.MessageSend

	if !ok {
		updateLogEntries(logId, webhookId, guildId, "WARNING: This event cannot be personalized, will try propogating to configured webhooks (if supported)?")

		var fields map[string]any

		if err := json.Unmarshal(bodyBytes, &fields); err != nil {
			updateLogEntries(logId, webhookId, guildId, "Error unmarshalling event: "+err.Error())
			state.Logger.Error("Error unmarshalling event", zap.Error(err), zap.String("repoName", rw.Repo.FullName), zap.String("webhookID", webhookId), zap.String("logId", logId))
			return
		}

		// Build a cleaner fallback embed instead of dumping raw map values
		providerLabel := "GitHub"
		if provider == "gitlab" {
			providerLabel = "GitLab"
		}

		var embed = discordgo.MessageEmbed{
			Title:       cases.Title(language.English).String(strings.ReplaceAll(header, "_", " ")),
			Description: "This webhook type is not fully customized yet; key payload fields are shown below.",
			Color:       0x8B949E,
			Timestamp:   time.Now().UTC().Format(time.RFC3339),
			URL:         rw.Repo.HTMLURL,
			Footer: &discordgo.MessageEmbedFooter{
				Text:    rw.Repo.FullName + " · " + providerLabel + " · Unhandled",
				IconURL: rw.Repo.Owner.AvatarURL,
			},
		}

		embed.Fields = buildFallbackWebhookEmbedFields(fields)

		messageSend = &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{&embed},
		}
	} else {
		// This event can be personalized
		updateLogEntries(logId, webhookId, guildId, "SUCCESS: This event can be personalized")
		messageSend, err = evtFn(bodyBytes)

		if err != nil {
			updateLogEntries(logId, webhookId, guildId, "Error processing event:", err.Error())
			state.Logger.Error("Error processing event", zap.Error(err), zap.String("repoName", rw.Repo.FullName), zap.String("webhookID", webhookId), zap.String("event", header), zap.String("logId", logId))
			return
		}
	}

	if messageSend == nil {
		updateLogEntries(logId, webhookId, guildId, "Error: event handler returned nil message")
		state.Logger.Error("Event handler returned nil messageSend", zap.String("repoName", rw.Repo.FullName), zap.String("webhookID", webhookId), zap.String("event", header), zap.String("logId", logId))
		return
	}

	events.EnrichEmbedsBeforeSend(messageSend.Embeds, rw, provider)

	for i, embed := range messageSend.Embeds {
		messageSend.Embeds[i] = applyEmbedLimits(embed)
	}

	for _, channelId := range channelIds {
		updateLogEntries(logId, webhookId, guildId, "Sending event to channel: channelId="+channelId)
		_, err := state.Discord.ChannelMessageSendComplex(channelId, messageSend)

		if err != nil {
			state.Discord.ChannelMessageSendComplex(channelId, &discordgo.MessageSend{
				Content: "Could not send event " + header + " to channel: <#" + channelId + ">:" + err.Error(),
			})

			updateLogEntries(logId, webhookId, guildId, "Could not send event "+header+" to channel: channelId="+channelId, "err="+err.Error())
		}
	}
}

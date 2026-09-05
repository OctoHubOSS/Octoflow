//  Copyright (C) 2026 NodeByte LTD

package ontos

import (
	"fmt"
	"time"

	"github.com/OctoHubOSS/Octoflow/webserver/state"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

const (
	deadWebhookCheckInterval = 24 * time.Hour
	deadWebhookAfter         = 14 * 24 * time.Hour
	renudgeCooldown          = 30 * 24 * time.Hour
)

func StartDeadWebhookChecker() {
	ticker := time.NewTicker(deadWebhookCheckInterval)
	defer ticker.Stop()

	checkDeadWebhooks()

	for range ticker.C {
		checkDeadWebhooks()
	}
}

type deadWebhookCandidate struct {
	id          string
	comment     string
	createdBy   string
	createdAt   time.Time
	lastEventAt *time.Time
}

func checkDeadWebhooks() {
	rows, err := state.Pool.Query(state.Context, `
		SELECT w.id, w.comment, w.created_by, w.created_at,
			(SELECT MAX(m.occurred_at) FROM `+state.TableEventMetrics+` m WHERE m.webhook_id = w.id) AS last_event_at
		FROM `+state.TableWebhooks+` w
		WHERE w.broken = false
			AND EXISTS (SELECT 1 FROM `+state.TableRepos+` r WHERE r.webhook_id = w.id)
			AND (w.last_nudged_at IS NULL OR w.last_nudged_at < NOW() - INTERVAL '30 days')
	`)
	if err != nil {
		state.Logger.Error("Failed to query webhooks for dead-webhook check", zap.Error(err))
		return
	}

	var candidates []deadWebhookCandidate
	for rows.Next() {
		var c deadWebhookCandidate
		if err := rows.Scan(&c.id, &c.comment, &c.createdBy, &c.createdAt, &c.lastEventAt); err != nil {
			state.Logger.Error("Failed to scan webhook row in dead-webhook check", zap.Error(err))
			continue
		}
		candidates = append(candidates, c)
	}
	rows.Close()

	for _, c := range candidates {
		reference := c.createdAt
		if c.lastEventAt != nil {
			reference = *c.lastEventAt
		}

		if time.Since(reference) < deadWebhookAfter {
			continue
		}

		nudgeDeadWebhook(c)
	}
}

func nudgeDeadWebhook(c deadWebhookCandidate) {
	channel, err := state.Discord.UserChannelCreate(c.createdBy)
	if err != nil {
		state.Logger.Warn("Could not open DM for dead-webhook nudge", zap.Error(err), zap.String("webhookID", c.id), zap.String("userID", c.createdBy))
		return
	}

	comment := c.comment
	if comment == "" {
		comment = "(no comment set)"
	}

	sinceText := "since it was created"
	if c.lastEventAt != nil {
		sinceText = "since its last event"
	}

	embed := &discordgo.MessageEmbed{
		Title: "One of your webhooks looks inactive",
		Description: fmt.Sprintf(
			"Your webhook **%s** (`%s`) hasn't received any GitHub events in over 14 days %s. "+
				"This usually means the payload URL or secret in the GitHub repo/org's webhook settings "+
				"is missing, wrong, or was removed.\n\n"+
				"Check it with `/list`, or rotate the secret with `/resetsecret` if you suspect it's out of sync. "+
				"If this webhook is intentionally quiet, there's nothing to do - you won't be nudged again for 30 days either way.",
			comment, c.id, sinceText,
		),
		Color: 0xfab219,
	}

	if _, err := state.Discord.ChannelMessageSendEmbed(channel.ID, embed); err != nil {
		state.Logger.Warn("Could not send dead-webhook nudge DM", zap.Error(err), zap.String("webhookID", c.id), zap.String("userID", c.createdBy))
		return
	}

	if _, err := state.Pool.Exec(state.Context, "UPDATE "+state.TableWebhooks+" SET last_nudged_at = NOW() WHERE id = $1", c.id); err != nil {
		state.Logger.Error("Failed to update last_nudged_at", zap.Error(err), zap.String("webhookID", c.id))
	}
}

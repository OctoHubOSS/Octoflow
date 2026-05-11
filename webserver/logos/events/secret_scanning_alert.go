package events

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

type SecretScanningAlertEvent struct {
	Action string     `json:"action"`
	Repo   Repository `json:"repository"`
	Sender User       `json:"sender"`
	Alert  struct {
		Number               int    `json:"number"`
		State                string `json:"state"`
		SecretType           string `json:"secret_type"`
		SecretTypeDisplayName string `json:"secret_type_display_name"`
		Secret               string `json:"secret"` // redacted by GitHub
		HTMLURL              string `json:"html_url"`
		CreatedAt            string `json:"created_at"`
		Resolution           string `json:"resolution"`
		ResolvedAt           string `json:"resolved_at"`
		ResolvedBy           User   `json:"resolved_by"`
		PushProtectionBypassed bool  `json:"push_protection_bypassed"`
		PushProtectionBypassedBy User `json:"push_protection_bypassed_by"`
	} `json:"alert"`
}

func secretScanningAlertFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh SecretScanningAlertEvent

	err := json.Unmarshal(bytes, &gh)
	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	color := colorRed
	switch gh.Action {
	case "resolved":
		color = colorGreen
	case "revoked":
		color = colorDarkRed
	}

	fields := []*discordgo.MessageEmbedField{
		{
			Name:   "Action",
			Value:  gh.Action,
			Inline: true,
		},
		{
			Name:   "State",
			Value:  gh.Alert.State,
			Inline: true,
		},
		{
			Name:   "Secret Type",
			Value:  gh.Alert.SecretTypeDisplayName,
			Inline: true,
		},
	}

	if gh.Alert.Resolution != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Resolution",
			Value:  gh.Alert.Resolution,
			Inline: true,
		})
	}

	if gh.Alert.ResolvedBy.Login != "" {
		resolveInfo := gh.Alert.ResolvedBy.Link()
		if gh.Alert.ResolvedAt != "" {
			if t, err := time.Parse(time.RFC3339, gh.Alert.ResolvedAt); err == nil {
				resolveInfo += " at " + t.Format("2006-01-02 15:04 UTC")
			}
		}
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:  "Resolved By",
			Value: resolveInfo,
		})
	}

	if gh.Alert.PushProtectionBypassed {
		bypassInfo := "Yes"
		if gh.Alert.PushProtectionBypassedBy.Login != "" {
			bypassInfo = "By " + gh.Alert.PushProtectionBypassedBy.Link()
		}
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:  "Push Protection Bypassed",
			Value: bypassInfo,
		})
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:  color,
				URL:    gh.Alert.HTMLURL,
				Author: gh.Sender.AuthorEmbed(),
				Title:  fmt.Sprintf("Secret Scanning Alert #%d %s on %s", gh.Alert.Number, gh.Action, gh.Repo.FullName),
				Fields: fields,
			},
		},
	}, nil
}

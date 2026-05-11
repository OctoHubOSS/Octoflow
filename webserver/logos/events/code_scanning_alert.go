package events

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

type CodeScanningAlertEvent struct {
	Action string     `json:"action"`
	Repo   Repository `json:"repository"`
	Sender User       `json:"sender"`
	Alert  struct {
		Number             int    `json:"number"`
		State              string `json:"state"`
		FixedAt            string `json:"fixed_at"`
		DismissedAt        string `json:"dismissed_at"`
		DismissedReason    string `json:"dismissed_reason"`
		DismissedComment   string `json:"dismissed_comment"`
		CreatedAt          string `json:"created_at"`
		HTMLURL            string `json:"html_url"`
		Rule               struct {
			ID                    string `json:"id"`
			Severity              string `json:"severity"`
			SecuritySeverityLevel string `json:"security_severity_level"`
			Description           string `json:"description"`
			FullDescription       string `json:"full_description"`
			Name                  string `json:"name"`
		} `json:"rule"`
		Tool struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"tool"`
		MostRecentInstance struct {
			Ref         string `json:"ref"`
			State       string `json:"state"`
			CommitSHA   string `json:"commit_sha"`
			Location    struct {
				Path        string `json:"path"`
				StartLine   int    `json:"start_line"`
				EndLine     int    `json:"end_line"`
				StartColumn int    `json:"start_column"`
				EndColumn   int    `json:"end_column"`
			} `json:"location"`
		} `json:"most_recent_instance"`
		DismissedBy User `json:"dismissed_by"`
	} `json:"alert"`
}

func codeScanningAlertFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh CodeScanningAlertEvent

	err := json.Unmarshal(bytes, &gh)
	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	color := colorYellow
	switch gh.Action {
	case "fixed":
		color = colorGreen
	case "appeared_in_branch", "created", "reopened":
		color = colorRed
	case "closed_by_user":
		color = colorDarkRed
	}

	// Build severity info
	severity := gh.Alert.Rule.Severity
	if gh.Alert.Rule.SecuritySeverityLevel != "" {
		severity = gh.Alert.Rule.SecuritySeverityLevel
	}
	if severity == "" {
		severity = "unknown"
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
			Name:   "Severity",
			Value:  severity,
			Inline: true,
		},
	}

	if gh.Alert.Rule.Name != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Rule",
			Value:  gh.Alert.Rule.Name + " (`" + gh.Alert.Rule.ID + "`)",
			Inline: false,
		})
	}

	if gh.Alert.Rule.Description != "" {
		desc := gh.Alert.Rule.Description
		if len(desc) > 200 {
			desc = desc[:200] + "..."
		}
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:  "Description",
			Value: desc,
		})
	}

	if gh.Alert.Tool.Name != "" {
		toolInfo := gh.Alert.Tool.Name
		if gh.Alert.Tool.Version != "" {
			toolInfo += " v" + gh.Alert.Tool.Version
		}
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Tool",
			Value:  toolInfo,
			Inline: true,
		})
	}

	loc := gh.Alert.MostRecentInstance.Location
	if loc.Path != "" {
		locStr := fmt.Sprintf("`%s` L%d", loc.Path, loc.StartLine)
		if loc.EndLine > loc.StartLine {
			locStr = fmt.Sprintf("`%s` L%d-L%d", loc.Path, loc.StartLine, loc.EndLine)
		}
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Location",
			Value:  locStr,
			Inline: true,
		})
	}

	if gh.Alert.MostRecentInstance.Ref != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Ref",
			Value:  gh.Alert.MostRecentInstance.Ref,
			Inline: true,
		})
	}

	if gh.Alert.DismissedBy.Login != "" {
		dismissInfo := "By: " + gh.Alert.DismissedBy.Link()
		if gh.Alert.DismissedReason != "" {
			dismissInfo += "\nReason: " + gh.Alert.DismissedReason
		}
		if gh.Alert.DismissedAt != "" {
			if t, err := time.Parse(time.RFC3339, gh.Alert.DismissedAt); err == nil {
				dismissInfo += "\nAt: " + t.Format("2006-01-02 15:04 UTC")
			}
		}
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:  "Dismissal",
			Value: dismissInfo,
		})
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:  color,
				URL:    gh.Alert.HTMLURL,
				Author: gh.Sender.AuthorEmbed(),
				Title:  fmt.Sprintf("Code Scanning Alert #%d %s on %s", gh.Alert.Number, gh.Action, gh.Repo.FullName),
				Fields: fields,
			},
		},
	}, nil
}

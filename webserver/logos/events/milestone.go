package events

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

type MilestoneEvent struct {
	Action    string     `json:"action"`
	Repo      Repository `json:"repository"`
	Sender    User       `json:"sender"`
	Milestone struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		HTMLURL     string `json:"html_url"`
		State       string `json:"state"`
		DueOn       string `json:"due_on"`
	} `json:"milestone"`
}

func milestoneFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh MilestoneEvent

	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	var color int
	if gh.Action == "deleted" || gh.Action == "closed" {
		color = colorRed
	} else {
		color = colorGreen
	}

	description := gh.Milestone.Description

	if len(description) > 1000 {
		description = description[:1000] + "..."
	}

	if description == "" {
		description = "No description provided."
	}

	dueOn := gh.Milestone.DueOn

	if dueOn == "" {
		dueOn = "No due date set."
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:       color,
				Timestamp:   time.Now().UTC().Format(time.RFC3339),
				URL:         gh.Milestone.HTMLURL,
				Author:      gh.Sender.AuthorEmbed(),
				Description: description,
				Title:       "Milestone " + gh.Action + " on " + gh.Repo.FullName,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Title",
						Value:  gh.Milestone.Title,
						Inline: true,
					},
					{
						Name:   "State",
						Value:  gh.Milestone.State,
						Inline: true,
					},
					{
						Name:  "Due",
						Value: dueOn,
					},
				},
			},
		},
	}, nil
}

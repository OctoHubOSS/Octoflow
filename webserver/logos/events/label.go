package events

import (
	"github.com/bwmarrin/discordgo"
)

type LabelEvent struct {
	Action string     `json:"action"`
	Repo   Repository `json:"repository"`
	Sender User       `json:"sender"`
	Label  struct {
		Name        string `json:"name"`
		Color       string `json:"color"`
		Description string `json:"description"`
	} `json:"label"`
}

func labelFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh LabelEvent

	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	var color int
	if gh.Action == "deleted" {
		color = colorRed
	} else {
		color = colorGreen
	}

	description := gh.Label.Description

	if description == "" {
		description = "No description provided."
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:     color,
				Timestamp: nowTimestamp(),
				URL:       gh.Repo.HTMLURL,
				Author:    gh.Sender.AuthorEmbed(),
				Title:     "Label " + gh.Action + " on " + gh.Repo.FullName,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Label",
						Value:  gh.Label.Name,
						Inline: true,
					},
					{
						Name:   "Color",
						Value:  "#" + gh.Label.Color,
						Inline: true,
					},
					{
						Name:  "Description",
						Value: description,
					},
				},
			},
		},
	}, nil
}

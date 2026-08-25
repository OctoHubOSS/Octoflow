package events

import (
	"time"

	"strings"

	"github.com/bwmarrin/discordgo"
)

type RepositoryEvent struct {
	Action string     `json:"action"`
	Repo   Repository `json:"repository"`
	Sender User       `json:"sender"`
}

func repositoryFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh RepositoryEvent

	// Unmarshal the JSON into our struct
	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	var color int
	var title string
	if gh.Action == "created" {
		color = colorGreen
		title = "Created: " + gh.Repo.FullName
	} else {
		color = colorRed
		title = strings.ToUpper(gh.Action) + ": " + gh.Repo.FullName
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:     color,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				URL:       gh.Repo.HTMLURL,
				Title:     title,
				Author:    gh.Sender.AuthorEmbed(),
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:  "User",
						Value: gh.Sender.Link(),
					},
				},
			},
		},
	}, nil
}

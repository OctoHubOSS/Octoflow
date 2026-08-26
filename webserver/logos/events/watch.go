package events

import (
	"github.com/bwmarrin/discordgo"
)

type WatchEvent struct {
	Action string     `json:"action"`
	Repo   Repository `json:"repository"`
	Sender User       `json:"sender"`
}

func watchFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh WatchEvent

	// Unmarshal the JSON into our struct
	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	var color int
	var title string
	color = colorGreen
	title = "Watch " + gh.Action + ": " + gh.Repo.FullName
	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:     color,
				Timestamp: nowTimestamp(),
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

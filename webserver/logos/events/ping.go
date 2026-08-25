package events

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

type PingEvent struct {
	Zen    string     `json:"zen"`
	Repo   Repository `json:"repository"`
	Sender User       `json:"sender"`
}

func pingFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh PingEvent

	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:       colorGreen,
				Timestamp:   time.Now().UTC().Format(time.RFC3339),
				URL:         gh.Repo.HTMLURL,
				Author:      gh.Sender.AuthorEmbed(),
				Title:       "Webhook connected to " + gh.Repo.FullName,
				Description: gh.Zen,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:  "Set up by",
						Value: gh.Sender.Link(),
					},
				},
			},
		},
	}, nil
}

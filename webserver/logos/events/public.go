package events

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type PublicEvent struct {
	Repo   Repository `json:"repository"`
	Sender User       `json:"sender"`
}

func publicFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh PublicEvent

	// Unmarshal the JSON into our struct
	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:     colorGreen,
				Timestamp: nowTimestamp(),
				URL:       gh.Repo.HTMLURL,
				Author:    gh.Sender.AuthorEmbed(),
				Title:     fmt.Sprintf("Repository update: %s", gh.Repo.FullName),
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:  "User",
						Value: gh.Sender.Link(),
					},
					{
						Name:  "Changes",
						Value: "private -> public",
					},
				},
			},
		},
	}, nil
}

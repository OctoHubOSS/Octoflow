package events

import (
	"time"

	"fmt"

	"github.com/bwmarrin/discordgo"
)

type ForkEvent struct {
	Action string     `json:"action"`
	Repo   Repository `json:"repository"`
	Forkee Repository `json:"forkee"`
	Sender User       `json:"sender"`
}

func forkFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh ForkEvent

	// Unmarshal the JSON into our struct
	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:     colorGreen,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				URL:       gh.Forkee.HTMLURL,
				Author:    gh.Sender.AuthorEmbed(),
				Title:     "New fork: " + gh.Forkee.FullName,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:  "User",
						Value: gh.Sender.Link(),
					},
					{
						Name:  "Original repo",
						Value: gh.Repo.HTMLURL,
					},
					{
						Name:  "Forked repo",
						Value: gh.Forkee.HTMLURL,
					},
					{
						Name:  "Visibility",
						Value: fmt.Sprintf("%s -> %s", gh.Repo.Visibility(), gh.Forkee.Visibility()),
					},
				},
			},
		},
	}, nil

}

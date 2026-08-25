package events

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

type DeleteEvent struct {
	Repo       Repository `json:"repository"`
	Sender     User       `json:"sender"`
	Ref        string     `json:"ref"`
	RefType    string     `json:"ref_type"`
	PusherType string     `json:"pusher_type"`
}

func deleteFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh DeleteEvent

	// Unmarshal the JSON into our struct
	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:     colorRed,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				URL:       gh.Repo.HTMLURL,
				Author:    gh.Sender.AuthorEmbed(),
				Title:     "Removed " + gh.RefType + " from " + gh.Repo.FullName,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:  "User",
						Value: gh.Sender.Link(),
					},
					{
						Name:   "Ref",
						Value:  gh.Ref,
						Inline: true,
					},
					{
						Name:   "Ref Type",
						Value:  gh.RefType,
						Inline: true,
					},
					{
						Name:   "Pusher Type",
						Value:  gh.PusherType,
						Inline: true,
					},
				},
			},
		},
	}, nil
}

package events

import (
	"github.com/bwmarrin/discordgo"
)

type StarEvent struct {
	Action string     `json:"action"`
	Repo   Repository `json:"repository"`
	Sender User       `json:"sender"`
}

func starFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh StarEvent

	// Unmarshal the JSON into our struct
	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	var color int
	var title string
	if gh.Action == "created" {
		color = colorGreen
		title = "Starred: " + gh.Repo.FullName
	} else {
		color = colorRed
		title = "Unstarred: " + gh.Repo.FullName
	}
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

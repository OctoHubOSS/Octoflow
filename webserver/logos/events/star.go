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

	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	var color int
	var title string
	var desc string
	if gh.Action == "created" {
		color = colorGreen
		title = "Star · " + gh.Repo.FullName
		desc = gh.Sender.Link() + " starred " + gh.Repo.MarkdownLink() + " ✨"
	} else {
		color = colorRed
		title = "Unstar · " + gh.Repo.FullName
		desc = gh.Sender.Link() + " removed their star from " + gh.Repo.MarkdownLink()
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:       color,
				URL:         gh.Repo.HTMLURL,
				Thumbnail:   gh.Sender.EmbedThumbnail(),
				Title:       title,
				Author:      gh.Sender.AuthorEmbed(),
				Description: desc,
			},
		},
	}, nil
}

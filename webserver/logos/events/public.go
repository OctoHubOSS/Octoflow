package events

import (
	"github.com/bwmarrin/discordgo"
)

type PublicEvent struct {
	Repo   Repository `json:"repository"`
	Sender User       `json:"sender"`
}

func publicFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh PublicEvent

	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	desc := "**" + gh.Repo.MarkdownLink() + "** is now **public**.\n\n" + gh.Sender.Link() + " flipped visibility from private → public."

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:       colorGreen,
				URL:         gh.Repo.HTMLURL,
				Thumbnail:   gh.Repo.OwnerThumbnail(),
				Title:       "Repository is public · " + gh.Repo.FullName,
				Author:      gh.Sender.AuthorEmbed(),
				Description: desc,
			},
		},
	}, nil
}

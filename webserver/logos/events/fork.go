package events

import (
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

	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	desc := "**Upstream:** " + gh.Repo.MarkdownLink() + "\n**Fork:** " + gh.Forkee.MarkdownLink()

	thumb := gh.Forkee.OwnerThumbnail()
	if thumb == nil {
		thumb = gh.Repo.OwnerThumbnail()
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:       colorGreen,
				URL:         gh.Forkee.HTMLURL,
				Thumbnail:   thumb,
				Author:      gh.Sender.AuthorEmbed(),
				Title:       "Fork · " + gh.Forkee.FullName,
				Description: desc,
				Fields: []*discordgo.MessageEmbedField{
					{Name: "Actor", Value: gh.Sender.Link(), Inline: false},
				},
			},
		},
	}, nil
}

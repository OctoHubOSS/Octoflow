package events

import (
	"github.com/bwmarrin/discordgo"
)

type MemberEvent struct {
	Action string     `json:"action"`
	Repo   Repository `json:"repository"`
	Sender User       `json:"sender"`
	Member User       `json:"member"`
}

func memberFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh MemberEvent

	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	var color int
	if gh.Action == "removed" {
		color = colorRed
	} else {
		color = colorGreen
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:     color,
				Timestamp: nowTimestamp(),
				URL:       gh.Repo.HTMLURL,
				Author:    gh.Sender.AuthorEmbed(),
				Title:     "Collaborator " + gh.Action + " on " + gh.Repo.FullName,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Member",
						Value:  gh.Member.Link(),
						Inline: true,
					},
					{
						Name:   "By",
						Value:  gh.Sender.Link(),
						Inline: true,
					},
				},
			},
		},
	}, nil
}

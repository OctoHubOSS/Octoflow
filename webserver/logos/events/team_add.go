package events

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type TeamAddEvent struct {
	Repo   Repository `json:"repository"`
	Sender User       `json:"sender"`
	Team   struct {
		Name    string `json:"name"`
		Slug    string `json:"slug"`
		HTMLUrl string `json:"html_url"`
	} `json:"team"`
}

func teamAddFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh TeamAddEvent

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
				Title:     "Team added to " + gh.Repo.FullName,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:  "Team",
						Value: fmt.Sprintf("[%s](%s)", gh.Team.Name, gh.Team.HTMLUrl),
					},
				},
			},
		},
	}, nil
}

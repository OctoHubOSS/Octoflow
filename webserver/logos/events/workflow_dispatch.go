package events

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

type WorkflowDispatchEvent struct {
	Ref      string     `json:"ref"`
	Workflow string     `json:"workflow"`
	Repo     Repository `json:"repository"`
	Sender   User       `json:"sender"`
}

func workflowDispatchFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh WorkflowDispatchEvent

	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:     colorGreen,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				URL:       gh.Repo.HTMLURL,
				Author:    gh.Sender.AuthorEmbed(),
				Title:     "Workflow manually triggered on " + gh.Repo.FullName,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Workflow",
						Value:  gh.Workflow,
						Inline: true,
					},
					{
						Name:   "Ref",
						Value:  gh.Ref,
						Inline: true,
					},
					{
						Name:  "Triggered by",
						Value: gh.Sender.Link(),
					},
				},
			},
		},
	}, nil
}

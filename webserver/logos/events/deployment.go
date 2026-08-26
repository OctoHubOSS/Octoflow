package events

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

type DeploymentEvent struct {
	Action     string     `json:"action"`
	Repo       Repository `json:"repository"`
	Sender     User       `json:"sender"`
	Deployment struct {
		Creator               User      `json:"creator"`
		CreatedAt             time.Time `json:"created_at"`
		SHA                   string    `json:"sha"`
		Description           string    `json:"description"`
		OriginalEnvironment   string    `json:"original_environment"`
		Environment           string    `json:"environment"`
		ProductionEnvironment bool      `json:"production_environment"`
		TransientEnvironment  bool      `json:"transient_environment"`
		StatusesUrl           string    `json:"statuses_url"`
	} `json:"deployment"`
}

func deploymentFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh DeploymentEvent

	// Unmarshal the JSON into our struct
	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	var color int
	var title string = "Deployment " + gh.Action + " on " + gh.Repo.FullName
	if gh.Action == "created" || gh.Action == "edited" {
		color = colorGreen
	} else {
		color = colorRed
	}

	var env = gh.Deployment.Environment

	if gh.Deployment.OriginalEnvironment != gh.Deployment.Environment && gh.Deployment.OriginalEnvironment != "" {
		env = gh.Deployment.OriginalEnvironment + " => " + gh.Deployment.Environment
	}

	gh.Deployment.Description = truncateText(gh.Deployment.Description, 996)

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:       color,
				URL:         gh.Repo.HTMLURL,
				Title:       title,
				Author:      gh.Deployment.Creator.AuthorEmbed(),
				Description: gh.Deployment.Description,
				Timestamp:   gh.Deployment.CreatedAt.Format(time.RFC3339),
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "User",
						Value:  gh.Sender.Link(),
						Inline: true,
					},
					{
						Name:   "Environment",
						Value:  env,
						Inline: true,
					},
					{
						Name:   "Commit",
						Value:  gh.Repo.Commit(gh.Deployment.SHA),
						Inline: true,
					},
					{
						Name:   "Is Production (according to github)",
						Value:  fmt.Sprintf("%t", gh.Deployment.ProductionEnvironment),
						Inline: true,
					},
					{
						Name:   "Is Transient Environment",
						Value:  fmt.Sprintf("%t", gh.Deployment.TransientEnvironment),
						Inline: true,
					},
				},
			},
		},
	}, nil
}

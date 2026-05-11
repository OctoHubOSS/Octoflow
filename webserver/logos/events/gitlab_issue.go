package events

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type GLIssueEvent struct {
	ObjectKind       string    `json:"object_kind"`
	EventType        string    `json:"event_type"`
	User             GLUser    `json:"user"`
	Project          GLProject `json:"project"`
	ObjectAttributes struct {
		ID          int    `json:"id"`
		IID         int    `json:"iid"`
		Title       string `json:"title"`
		Description string `json:"description"`
		State       string `json:"state"`
		Action      string `json:"action"`
		URL         string `json:"url"`
	} `json:"object_attributes"`
}

func glIssueFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gl GLIssueEvent
	err := json.Unmarshal(bytes, &gl)
	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	body := gl.ObjectAttributes.Description
	if len(body) > 996 {
		body = body[:996] + "..."
	}
	if body == "" {
		body = "No description available"
	}

	color := colorGreen
	if gl.ObjectAttributes.Action == "close" {
		color = colorRed
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:       color,
				URL:         gl.ObjectAttributes.URL,
				Author:      gl.User.AuthorEmbed(),
				Description: body,
				Title:       fmt.Sprintf("Issue %s on %s (#%d)", gl.ObjectAttributes.Action, gl.Project.PathWithNamespace, gl.ObjectAttributes.IID),
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Action",
						Value:  gl.ObjectAttributes.Action,
						Inline: true,
					},
					{
						Name:   "State",
						Value:  gl.ObjectAttributes.State,
						Inline: true,
					},
					{
						Name:   "Title",
						Value:  gl.ObjectAttributes.Title,
						Inline: true,
					},
				},
				Footer: &discordgo.MessageEmbedFooter{
					Text: "GitLab",
				},
			},
		},
	}, nil
}

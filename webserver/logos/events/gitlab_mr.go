package events

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type GLMergeRequestEvent struct {
	ObjectKind       string    `json:"object_kind"`
	EventType        string    `json:"event_type"`
	User             GLUser    `json:"user"`
	Project          GLProject `json:"project"`
	ObjectAttributes struct {
		ID                        int    `json:"id"`
		IID                       int    `json:"iid"`
		Title                     string `json:"title"`
		Description               string `json:"description"`
		State                     string `json:"state"`
		Action                    string `json:"action"`
		URL                       string `json:"url"`
		SourceBranch              string `json:"source_branch"`
		TargetBranch              string `json:"target_branch"`
		MergeStatus               string `json:"merge_status"`
		MergeWhenPipelineSucceeds bool   `json:"merge_when_pipeline_succeeds"`
	} `json:"object_attributes"`
}

func glMergeRequestFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gl GLMergeRequestEvent
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

	color := glColorPurple
	if gl.ObjectAttributes.Action == "merge" {
		color = colorGreen
	} else if gl.ObjectAttributes.Action == "close" {
		color = colorRed
	}

	desc := fmt.Sprintf("_Action:_ **`%s`**\n\n%s", gl.ObjectAttributes.Action, body)

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:       color,
				URL:         gl.ObjectAttributes.URL,
				Thumbnail:   glProjectThumbnail(gl.Project),
				Author:      gl.User.AuthorEmbed(),
				Description: desc,
				Title:       fmt.Sprintf("Merge request · %s · !%d", gl.Project.PathWithNamespace, gl.ObjectAttributes.IID),
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Title",
						Value:  gl.ObjectAttributes.Title,
						Inline: false,
					},
					{
						Name:   "Branches",
						Value:  "`" + gl.ObjectAttributes.SourceBranch + "` → `" + gl.ObjectAttributes.TargetBranch + "`",
						Inline: false,
					},
					{
						Name:   "Merge status",
						Value:  "`" + gl.ObjectAttributes.MergeStatus + "`",
						Inline: true,
					},
					{
						Name:   "State",
						Value:  "`" + gl.ObjectAttributes.State + "`",
						Inline: true,
					},
				},
				Footer: glFooterForProject(gl.Project),
			},
		},
	}, nil
}

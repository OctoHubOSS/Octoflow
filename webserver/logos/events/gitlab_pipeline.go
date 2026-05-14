package events

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type GLPipelineEvent struct {
	ObjectKind       string    `json:"object_kind"`
	User             GLUser    `json:"user"`
	Project          GLProject `json:"project"`
	ObjectAttributes struct {
		ID         int      `json:"id"`
		IID        int      `json:"iid"`
		Ref        string   `json:"ref"`
		Status     string   `json:"status"`
		Source     string   `json:"source"`
		Stages     []string `json:"stages"`
		Duration   int      `json:"duration"`
		CreatedAt  string   `json:"created_at"`
		FinishedAt string   `json:"finished_at"`
	} `json:"object_attributes"`
	Builds []struct {
		ID        int    `json:"id"`
		Stage     string `json:"stage"`
		Name      string `json:"name"`
		Status    string `json:"status"`
		CreatedAt string `json:"created_at"`
	} `json:"builds"`
}

func glPipelineFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gl GLPipelineEvent
	err := json.Unmarshal(bytes, &gl)
	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	color := colorYellow
	switch gl.ObjectAttributes.Status {
	case "success":
		color = colorGreen
	case "failed":
		color = colorRed
	case "canceled", "skipped":
		color = colorDarkRed
	}

	stages := strings.Join(gl.ObjectAttributes.Stages, " → ")
	if stages == "" {
		stages = "N/A"
	}

	var buildSummary string
	for _, b := range gl.Builds {
		icon := "⏳"
		switch b.Status {
		case "success":
			icon = "✅"
		case "failed":
			icon = "❌"
		case "skipped":
			icon = "⏭️"
		case "canceled":
			icon = "🚫"
		case "running":
			icon = "🔄"
		}
		buildSummary += fmt.Sprintf("%s %s (%s)\n", icon, b.Name, b.Stage)
	}

	if len(buildSummary) > 1020 {
		buildSummary = buildSummary[:1020] + "..."
	}
	if buildSummary == "" {
		buildSummary = "No builds"
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:  color,
				URL:    gl.Project.WebURL + "/-/pipelines/" + fmt.Sprintf("%d", gl.ObjectAttributes.ID),
				Author: gl.User.AuthorEmbed(),
				Title:  fmt.Sprintf("Pipeline · #%d · %s · %s", gl.ObjectAttributes.ID, gl.ObjectAttributes.Status, gl.Project.PathWithNamespace),
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Ref",
						Value:  gl.ObjectAttributes.Ref,
						Inline: true,
					},
					{
						Name:   "Status",
						Value:  gl.ObjectAttributes.Status,
						Inline: true,
					},
					{
						Name:   "Source",
						Value:  gl.ObjectAttributes.Source,
						Inline: true,
					},
					{
						Name:  "Stages",
						Value: stages,
					},
					{
						Name:  "Jobs",
						Value: buildSummary,
					},
				},
				Footer: &discordgo.MessageEmbedFooter{
					Text: "GitLab",
				},
			},
		},
	}, nil
}

type GLJobEvent struct {
	ObjectKind      string `json:"object_kind"`
	Ref             string `json:"ref"`
	BuildID         int    `json:"build_id"`
	BuildName       string `json:"build_name"`
	BuildStage      string `json:"build_stage"`
	BuildStatus     string `json:"build_status"`
	BuildDuration   float64 `json:"build_duration"`
	BuildFailureReason string `json:"build_failure_reason"`
	PipelineID      int    `json:"pipeline_id"`
	User            GLUser `json:"user"`
	Repository      GLRepository `json:"repository"`
	ProjectName     string `json:"project_name"`
}

func glJobFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gl GLJobEvent
	err := json.Unmarshal(bytes, &gl)
	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	color := colorYellow
	switch gl.BuildStatus {
	case "success":
		color = colorGreen
	case "failed":
		color = colorRed
	}

	fields := []*discordgo.MessageEmbedField{
		{
			Name:   "Job",
			Value:  gl.BuildName,
			Inline: true,
		},
		{
			Name:   "Stage",
			Value:  gl.BuildStage,
			Inline: true,
		},
		{
			Name:   "Status",
			Value:  gl.BuildStatus,
			Inline: true,
		},
	}

	if gl.BuildDuration > 0 {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Duration",
			Value:  fmt.Sprintf("%.1fs", gl.BuildDuration),
			Inline: true,
		})
	}

	if gl.BuildFailureReason != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:  "Failure Reason",
			Value: gl.BuildFailureReason,
		})
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:  color,
				Author: gl.User.AuthorEmbed(),
				Title:  fmt.Sprintf("Job · #%d · %s · %s", gl.BuildID, gl.BuildStatus, gl.ProjectName),
				Fields: fields,
				Footer: &discordgo.MessageEmbedFooter{
					Text: "GitLab",
				},
			},
		},
	}, nil
}

package events

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type GLReleaseEvent struct {
	ObjectKind string    `json:"object_kind"`
	Project    GLProject `json:"project"`
	Tag        string    `json:"tag"`
	Name       string    `json:"name"`
	Description string   `json:"description"`
	URL        string    `json:"url"`
	Action     string    `json:"action"`
	Assets     struct {
		Count int `json:"count"`
		Links []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"links"`
	} `json:"assets"`
}

func glReleaseFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gl GLReleaseEvent
	err := json.Unmarshal(bytes, &gl)
	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	desc := gl.Description
	if len(desc) > 996 {
		desc = desc[:996] + "..."
	}
	if desc == "" {
		desc = "No description"
	}

	color := colorGreen
	if gl.Action == "delete" {
		color = colorRed
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:       color,
				URL:         gl.URL,
				Description: desc,
				Title:       fmt.Sprintf("Release %s on %s (%s)", gl.Action, gl.Project.PathWithNamespace, gl.Tag),
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Name",
						Value:  gl.Name,
						Inline: true,
					},
					{
						Name:   "Tag",
						Value:  gl.Tag,
						Inline: true,
					},
					{
						Name:   "Assets",
						Value:  fmt.Sprintf("%d", gl.Assets.Count),
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

type GLWikiEvent struct {
	ObjectKind       string    `json:"object_kind"`
	User             GLUser    `json:"user"`
	Project          GLProject `json:"project"`
	Wiki             struct {
		WebURL string `json:"web_url"`
	} `json:"wiki"`
	ObjectAttributes struct {
		Title   string `json:"title"`
		Content string `json:"content"`
		Format  string `json:"format"`
		Slug    string `json:"slug"`
		URL     string `json:"url"`
		Action  string `json:"action"`
	} `json:"object_attributes"`
}

func glWikiFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gl GLWikiEvent
	err := json.Unmarshal(bytes, &gl)
	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	color := colorGreen
	if gl.ObjectAttributes.Action == "delete" {
		color = colorRed
	} else if gl.ObjectAttributes.Action == "update" {
		color = colorYellow
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:  color,
				URL:    gl.ObjectAttributes.URL,
				Author: gl.User.AuthorEmbed(),
				Title:  fmt.Sprintf("Wiki Page %s on %s", gl.ObjectAttributes.Action, gl.Project.PathWithNamespace),
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Title",
						Value:  gl.ObjectAttributes.Title,
						Inline: true,
					},
					{
						Name:   "Action",
						Value:  gl.ObjectAttributes.Action,
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

type GLDeploymentEvent struct {
	ObjectKind           string `json:"object_kind"`
	Status               string `json:"status"`
	StatusChangedAt      string `json:"status_changed_at"`
	DeployableID         int    `json:"deployable_id"`
	DeployableURL        string `json:"deployable_url"`
	Environment          string `json:"environment"`
	EnvironmentExternalURL string `json:"environment_external_url"`
	Project              GLProject `json:"project"`
	User                 GLUser    `json:"user"`
	ShortSHA             string    `json:"short_sha"`
	CommitURL            string    `json:"commit_url"`
	CommitTitle          string    `json:"commit_title"`
}

func glDeploymentFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gl GLDeploymentEvent
	err := json.Unmarshal(bytes, &gl)
	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	color := colorYellow
	switch gl.Status {
	case "success":
		color = colorGreen
	case "failed":
		color = colorRed
	}

	fields := []*discordgo.MessageEmbedField{
		{
			Name:   "Environment",
			Value:  gl.Environment,
			Inline: true,
		},
		{
			Name:   "Status",
			Value:  gl.Status,
			Inline: true,
		},
		{
			Name:   "Commit",
			Value:  fmt.Sprintf("[%s](%s)", gl.ShortSHA, gl.CommitURL),
			Inline: true,
		},
	}

	if gl.CommitTitle != "" {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:  "Commit Message",
			Value: gl.CommitTitle,
		})
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:  color,
				Author: gl.User.AuthorEmbed(),
				Title:  fmt.Sprintf("Deployment %s on %s", gl.Status, gl.Project.PathWithNamespace),
				Fields: fields,
				Footer: &discordgo.MessageEmbedFooter{
					Text: "GitLab",
				},
			},
		},
	}, nil
}

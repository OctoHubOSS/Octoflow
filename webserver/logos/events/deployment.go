package events

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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

	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	actionLabel := cases.Title(language.English).String(strings.ReplaceAll(gh.Action, "_", " "))

	color := colorGreen
	if gh.Action != "created" && gh.Action != "edited" {
		color = colorRed
	}

	env := gh.Deployment.Environment
	if gh.Deployment.OriginalEnvironment != gh.Deployment.Environment && gh.Deployment.OriginalEnvironment != "" {
		env = "`" + gh.Deployment.OriginalEnvironment + "` → `" + gh.Deployment.Environment + "`"
	} else {
		env = "`" + env + "`"
	}

	body := strings.TrimSpace(gh.Deployment.Description)
	if len(body) > 1200 {
		body = body[:1197] + "…"
	}
	if body == "" {
		body = "_No deployment description._"
	}

	desc := "**" + actionLabel + "** · " + gh.Repo.MarkdownLink() + "\n\n" + body
	if gh.Deployment.StatusesUrl != "" {
		desc += "\n\n[**Statuses**](" + gh.Deployment.StatusesUrl + ")"
	}

	ts := ""
	if !gh.Deployment.CreatedAt.IsZero() {
		ts = gh.Deployment.CreatedAt.UTC().Format(time.RFC3339)
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:       color,
				URL:         gh.Repo.HTMLURL,
				Thumbnail:   gh.Repo.OwnerThumbnail(),
				Author:      gh.Deployment.Creator.AuthorEmbed(),
				Title:       "Deployment · " + gh.Repo.FullName,
				Description: desc,
				Timestamp:   ts,
				Fields: []*discordgo.MessageEmbedField{
					{Name: "Environment", Value: env, Inline: false},
					{Name: "Commit", Value: gh.Repo.Commit(gh.Deployment.SHA), Inline: true},
					{Name: "Production", Value: fmt.Sprintf("`%t`", gh.Deployment.ProductionEnvironment), Inline: true},
					{Name: "Transient", Value: fmt.Sprintf("`%t`", gh.Deployment.TransientEnvironment), Inline: true},
					{Name: "Webhook actor", Value: gh.Sender.Link(), Inline: false},
				},
			},
		},
	}, nil
}

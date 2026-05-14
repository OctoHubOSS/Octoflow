package events

import (
	"strings"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type RepositoryEvent struct {
	Action string     `json:"action"`
	Repo   Repository `json:"repository"`
	Sender User       `json:"sender"`
}

func repositoryFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh RepositoryEvent

	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	actionLabel := cases.Title(language.English).String(strings.ReplaceAll(gh.Action, "_", " "))

	color := colorGreen
	if gh.Action != "created" {
		color = colorYellow
	}
	if gh.Action == "deleted" {
		color = colorRed
	}

	desc := "**" + actionLabel + "** · " + gh.Repo.MarkdownLink()
	if strings.TrimSpace(gh.Repo.Description) != "" {
		d := gh.Repo.Description
		if len(d) > 400 {
			d = d[:397] + "…"
		}
		desc += "\n\n_" + d + "_"
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:       color,
				URL:         gh.Repo.HTMLURL,
				Thumbnail:   gh.Repo.OwnerThumbnail(),
				Title:       "Repository · " + gh.Repo.FullName,
				Author:      gh.Sender.AuthorEmbed(),
				Description: desc,
				Fields: []*discordgo.MessageEmbedField{
					{Name: "Actor", Value: gh.Sender.Link(), Inline: true},
				},
			},
		},
	}, nil
}

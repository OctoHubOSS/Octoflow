package events

import (
	"strings"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type WatchEvent struct {
	Action string     `json:"action"`
	Repo   Repository `json:"repository"`
	Sender User       `json:"sender"`
}

func watchFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh WatchEvent

	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	actionLabel := cases.Title(language.English).String(strings.ReplaceAll(gh.Action, "_", " "))
	color := colorGreen
	desc := gh.Sender.Link() + " · _" + actionLabel + "_ on " + gh.Repo.MarkdownLink()

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:       color,
				URL:         gh.Repo.HTMLURL,
				Thumbnail:   gh.Repo.OwnerThumbnail(),
				Title:       "Watch · " + gh.Repo.FullName,
				Author:      gh.Sender.AuthorEmbed(),
				Description: desc,
			},
		},
	}, nil
}

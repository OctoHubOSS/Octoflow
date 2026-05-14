package events

import (
	"strings"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type ReleaseEvent struct {
	Action  string     `json:"action"`
	Repo    Repository `json:"repository"`
	Sender  User       `json:"sender"`
	Release struct {
		HTMLUrl string `json:"html_url"`
		Body    string `json:"body"`
		TagName string `json:"tag_name"`
	} `json:"release"`
}

func releaseFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh ReleaseEvent

	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	actionLabel := cases.Title(language.English).String(strings.ReplaceAll(gh.Action, "_", " "))

	color := colorGreen
	if gh.Action == "deleted" || gh.Action == "unpublished" {
		color = colorRed
	} else if gh.Action == "edited" || gh.Action == "prereleased" {
		color = colorYellow
	}

	body := strings.TrimSpace(gh.Release.Body)
	if len(body) > 1800 {
		body = body[:1797] + "…"
	}
	if body == "" {
		body = "_No release notes._"
	}

	page := strings.TrimSpace(gh.Release.HTMLUrl)
	if page == "" {
		page = gh.Repo.HTMLURL
	}

	tag := strings.TrimSpace(gh.Release.TagName)
	if tag == "" {
		tag = "Release"
	}

	desc := "**" + actionLabel + "** · [`" + tag + "`](" + page + ")\n\n" + body

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:       color,
				URL:         page,
				Thumbnail:   gh.Repo.OwnerThumbnail(),
				Title:       "Release · " + gh.Repo.FullName,
				Author:      gh.Sender.AuthorEmbed(),
				Description: desc,
				Fields: []*discordgo.MessageEmbedField{
					{Name: "Repository", Value: gh.Repo.MarkdownLink(), Inline: false},
					{Name: "Actor", Value: gh.Sender.Link(), Inline: true},
					{Name: "Tag", Value: "`" + gh.Release.TagName + "`", Inline: true},
				},
			},
		},
	}, nil
}

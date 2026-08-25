package events

import (
	"time"

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

	// Unmarshal the JSON into our struct
	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	var color int
	var title string = cases.Title(language.English).String(gh.Action) + " release on " + gh.Repo.FullName
	if gh.Action == "created" || gh.Action == "published" || gh.Action == "edited" || gh.Action == "prereleased" || gh.Action == "released" {
		color = colorGreen
	} else {
		color = colorRed
	}

	var body string = gh.Release.Body
	if len(gh.Release.Body) > 996 {
		body = gh.Release.Body[:996] + "..."
	}

	if body == "" {
		body = "No description available"
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:       color,
				Timestamp:   time.Now().UTC().Format(time.RFC3339),
				URL:         gh.Repo.HTMLURL,
				Title:       title,
				Author:      gh.Sender.AuthorEmbed(),
				Description: body,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "User",
						Value:  gh.Sender.Link(),
						Inline: true,
					},
					{
						Name:   "Release",
						Value:  "[" + gh.Release.TagName + "]" + "(" + gh.Release.HTMLUrl + ")",
						Inline: true,
					},
				},
			},
		},
	}, nil
}

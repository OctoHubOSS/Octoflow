package events

import (
	"time"

	"strings"

	"github.com/bwmarrin/discordgo"
)

type GollumEvent struct {
	Repo   Repository `json:"repository"`
	Sender User       `json:"sender"`
	Pages  []struct {
		PageName string `json:"page_name"`
		Title    string `json:"title"`
		Summary  string `json:"summary"`
		Action   string `json:"action"`
		HTMLURL  string `json:"html_url"`
	} `json:"pages"`
}

func gollumFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh GollumEvent

	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	var fields []*discordgo.MessageEmbedField

	for _, page := range gh.Pages {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:  page.Title + " (" + page.Action + ")",
			Value: page.HTMLURL,
		})
	}

	pageNames := make([]string, len(gh.Pages))
	for i, page := range gh.Pages {
		pageNames[i] = page.Title
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:     colorGreen,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				URL:       gh.Repo.HTMLURL,
				Author:    gh.Sender.AuthorEmbed(),
				Title:     "Wiki updated on " + gh.Repo.FullName + ": " + strings.Join(pageNames, ", "),
				Fields:    fields,
			},
		},
	}, nil
}

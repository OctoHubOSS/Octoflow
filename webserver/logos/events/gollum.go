package events

import (
	"fmt"

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
			Name:  truncateText(page.Title+" ("+page.Action+")", 256),
			Value: page.HTMLURL,
		})
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:     colorGreen,
				Timestamp: nowTimestamp(),
				URL:       gh.Repo.HTMLURL,
				Author:    gh.Sender.AuthorEmbed(),
				Title:     truncateText(fmt.Sprintf("Wiki updated on %s (%d page(s))", gh.Repo.FullName, len(gh.Pages)), 256),
				Fields:    fields,
			},
		},
	}, nil
}

package events

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type StatusEvent struct {
	Repo        Repository `json:"repository"`
	Sender      User       `json:"sender"`
	State       string     `json:"state"`
	Description string     `json:"description"`
	TargetURL   string     `json:"target_url"`
	Context     string     `json:"context"`
	Commit      struct {   // status
		HTMLURL string `json:"html_url"`
		SHA     string `json:"sha"`
		Author  struct {
			Login   string `json:"login"`
			HTMLURL string `json:"html_url"` // user
		} `json:"author"`
		Commit struct {
			ID      string `json:"id"`
			Message string `json:"message"`
		} `json:"commit"`
	} `json:"commit"`
}

func statusFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh StatusEvent

	// Unmarshal the JSON into our struct
	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	var moreInfoMsg string
	if gh.TargetURL != "" {
		moreInfoMsg = "\n\nFor more information, " + gh.TargetURL
	}

	if gh.Context == "" {
		gh.Context = "-"
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:       colorGreen,
				Timestamp:   nowTimestamp(),
				URL:         gh.Repo.HTMLURL,
				Author:      gh.Sender.AuthorEmbed(),
				Title:       "Status " + gh.State + " on " + gh.Repo.FullName,
				Description: gh.Description + moreInfoMsg,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:  "Commit",
						Value: fmt.Sprintf("[``%s``](%s) - %s | [%s](%s)", gh.Commit.SHA[:7], gh.Commit.HTMLURL, gh.Commit.Commit.Message, gh.Commit.Author.Login, gh.Commit.Author.HTMLURL),
					},
					{
						Name:   "User",
						Value:  gh.Sender.Link(),
						Inline: true,
					},
					{
						Name:   "Context",
						Value:  gh.Context,
						Inline: true,
					},
				},
			},
		},
	}, nil
}

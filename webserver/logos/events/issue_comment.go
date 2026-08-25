package events

import (
	"time"

	"fmt"

	"github.com/bwmarrin/discordgo"
)

type IssueCommentEvent struct {
	Action  string     `json:"action"`
	Repo    Repository `json:"repository"`
	Sender  User       `json:"sender"`
	Issue   Issue      `json:"issue"`
	Comment struct {
		Body    string `json:"body"`
		HTMLURL string `json:"html_url"`
		User    User   `json:"user"`
	} `json:"comment"`
}

func issueCommentFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh IssueCommentEvent

	// Unmarshal the JSON into our struct
	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	var body string = gh.Issue.Body
	if len(gh.Issue.Body) > 1000 {
		body = gh.Issue.Body[:1000] + "..."
	}

	if body == "" {
		body = "No description available"
	}

	var comment string = gh.Comment.Body

	if len(gh.Comment.Body) > 1000 {
		comment = gh.Comment.Body[:1000] + "..."
	}

	if comment == "" {
		comment = "No description available"
	}

	var color int
	if gh.Action == "deleted" {
		color = colorRed
	} else {
		color = colorGreen
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:     color,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				URL:       gh.Issue.HTMLURL,
				Author:    gh.Sender.AuthorEmbed(),
				Title:     fmt.Sprintf("Comment on %s (#%d) %s", gh.Repo.FullName, gh.Issue.Number, gh.Action),
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:  "User",
						Value: gh.Sender.Link(),
					},
					{
						Name:  "Title",
						Value: gh.Issue.Title,
					},
					{
						Name:  "Parent Issue",
						Value: body,
					},
					{
						Name:  "Comment",
						Value: comment,
					},
				},
			},
		},
	}, nil
}

package events

import (
	"strconv"

	"github.com/bwmarrin/discordgo"
)

type PullRequestReviewCommentEvent struct {
	Action      string      `json:"action"`
	Repo        Repository  `json:"repository"`
	Sender      User        `json:"sender"`
	PullRequest PullRequest `json:"pull_request"`
	Comment     struct {
		Body    string `json:"body"`
		HTMLURL string `json:"html_url"`
		User    User   `json:"user"`
	} `json:"comment"`
}

func pullRequestReviewCommentFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh PullRequestReviewCommentEvent

	// Unmarshal the JSON into our struct
	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	body := truncateText(gh.PullRequest.Body, 1000)

	if body == "" {
		body = "No description available"
	}

	comment := truncateText(gh.Comment.Body, 1000)

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
				Color:       color,
				Timestamp:   nowTimestamp(),
				URL:         gh.PullRequest.HTMLURL,
				Author:      gh.Sender.AuthorEmbed(),
				Description: comment,
				Title:       "Pull Request Review Comment on " + gh.Repo.FullName + " (#" + strconv.Itoa(gh.PullRequest.Number) + ")",
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:  "User",
						Value: gh.Comment.User.Link(),
					},
					{
						Name:  "Title",
						Value: gh.PullRequest.Title,
					},
					{
						Name:  "Parent Issue",
						Value: body,
					},
				},
			},
		},
	}, nil
}

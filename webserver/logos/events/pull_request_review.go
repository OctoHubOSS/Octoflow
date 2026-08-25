package events

import (
	"time"

	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type PullRequestReviewEvent struct {
	Action      string      `json:"action"`
	Repo        Repository  `json:"repository"`
	Sender      User        `json:"sender"`
	PullRequest PullRequest `json:"pull_request"`
	Review      struct {
		Body    string `json:"body"`
		HTMLURL string `json:"html_url"`
		State   string `json:"state"`
		User    User   `json:"user"`
	} `json:"review"`
}

func pullRequestReviewFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh PullRequestReviewEvent

	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	var body string = gh.Review.Body

	if len(body) > 1000 {
		body = body[:1000] + "..."
	}

	if body == "" {
		body = "No review comment left"
	}

	var color int
	switch gh.Review.State {
	case "approved":
		color = colorGreen
	case "changes_requested":
		color = colorRed
	default:
		color = colorYellow
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:       color,
				Timestamp:   time.Now().UTC().Format(time.RFC3339),
				URL:         gh.Review.HTMLURL,
				Author:      gh.Sender.AuthorEmbed(),
				Description: body,
				Title:       "Pull Request Review (" + strings.ReplaceAll(gh.Review.State, "_", " ") + ") on " + gh.Repo.FullName + " (#" + strconv.Itoa(gh.PullRequest.Number) + ")",
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Reviewer",
						Value:  gh.Review.User.Link(),
						Inline: true,
					},
					{
						Name:   "State",
						Value:  gh.Review.State,
						Inline: true,
					},
					{
						Name:  "Pull Request",
						Value: gh.PullRequest.Title,
					},
				},
			},
		},
	}, nil
}

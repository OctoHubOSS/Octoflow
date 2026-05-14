package events

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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

	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	prBody := strings.TrimSpace(gh.PullRequest.Body)
	if len(prBody) > 400 {
		prBody = prBody[:397] + "…"
	}
	if prBody == "" {
		prBody = "_No PR body._"
	}

	comment := strings.TrimSpace(gh.Comment.Body)
	if len(comment) > 1200 {
		comment = comment[:1197] + "…"
	}
	if comment == "" {
		comment = "_Empty review comment._"
	}

	color := colorGreen
	if gh.Action == "deleted" {
		color = colorRed
	}

	actionLabel := cases.Title(language.English).String(strings.ReplaceAll(gh.Action, "_", " "))

	page := strings.TrimSpace(gh.Comment.HTMLURL)
	if page == "" {
		page = gh.PullRequest.HTMLURL
	}

	desc := fmt.Sprintf("**%s** on [#%d — %s](%s)\n\n**Review comment**\n%s",
		actionLabel, gh.PullRequest.Number, gh.PullRequest.Title, gh.PullRequest.HTMLURL, comment)
	desc += "\n\n**PR excerpt**\n" + prBody

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:       color,
				URL:         page,
				Thumbnail:   gh.Sender.EmbedThumbnail(),
				Author:      gh.Sender.AuthorEmbed(),
				Title:       "PR review comment · " + gh.Repo.FullName + " · #" + strconv.Itoa(gh.PullRequest.Number),
				Description: desc,
				Fields: []*discordgo.MessageEmbedField{
					{Name: "Comment author", Value: gh.Comment.User.Link(), Inline: true},
					{Name: "Webhook actor", Value: gh.Sender.Link(), Inline: true},
				},
			},
		},
	}, nil
}

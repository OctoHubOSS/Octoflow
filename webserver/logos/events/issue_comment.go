package events

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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

	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	issueBody := strings.TrimSpace(gh.Issue.Body)
	if len(issueBody) > 400 {
		issueBody = issueBody[:397] + "…"
	}
	if issueBody == "" {
		issueBody = "_No issue body._"
	}

	comment := strings.TrimSpace(gh.Comment.Body)
	if len(comment) > 1200 {
		comment = comment[:1197] + "…"
	}
	if comment == "" {
		comment = "_Empty comment._"
	}

	color := colorGreen
	if gh.Action == "deleted" {
		color = colorRed
	}

	actionLabel := cases.Title(language.English).String(strings.ReplaceAll(gh.Action, "_", " "))

	page := strings.TrimSpace(gh.Comment.HTMLURL)
	if page == "" {
		page = gh.Issue.HTMLURL
	}

	desc := fmt.Sprintf("**%s** on [#%d — %s](%s)\n\n**Comment**\n%s",
		actionLabel, gh.Issue.Number, gh.Issue.Title, gh.Issue.HTMLURL, comment)
	desc += "\n\n**Issue excerpt**\n" + issueBody

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:       color,
				URL:         page,
				Thumbnail:   gh.Sender.EmbedThumbnail(),
				Author:      gh.Sender.AuthorEmbed(),
				Title:       "Issue comment · " + gh.Repo.FullName,
				Description: desc,
				Fields: []*discordgo.MessageEmbedField{
					{Name: "Comment author", Value: gh.Comment.User.Link(), Inline: true},
					{Name: "Webhook actor", Value: gh.Sender.Link(), Inline: true},
				},
			},
		},
	}, nil
}

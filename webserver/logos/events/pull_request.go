package events

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type PullRequestEvent struct {
	Action      string      `json:"action"`
	Repo        Repository  `json:"repository"`
	Sender      User        `json:"sender"`
	PullRequest PullRequest `json:"pull_request"`
}

func pullRequestFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh PullRequestEvent

	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	body := strings.TrimSpace(gh.PullRequest.Body)
	if len(body) > 1800 {
		body = body[:1797] + "…"
	}
	if body == "" {
		body = "_No description._"
	}

	color := colorGreen
	if gh.Action == "closed" {
		color = colorRed
	} else if gh.Action == "edited" || gh.Action == "labeled" || gh.Action == "unlabeled" || gh.Action == "synchronize" {
		color = colorYellow
	}

	actionLabel := cases.Title(language.English).String(strings.ReplaceAll(gh.Action, "_", " "))

	desc := fmt.Sprintf("**%s** · #%d · **`%s`**\n\n%s", actionLabel, gh.PullRequest.Number, gh.PullRequest.State, body)

	branches := fmt.Sprintf("`%s` ← `%s`", gh.PullRequest.Base.Ref, gh.PullRequest.Head.Ref)
	if gh.PullRequest.Base.Repo.FullName != "" || gh.PullRequest.Head.Repo.FullName != "" {
		branches += fmt.Sprintf("\n%s into %s", gh.PullRequest.Head.Repo.MarkdownLink(), gh.PullRequest.Base.Repo.MarkdownLink())
	}

	thumb := gh.PullRequest.User.EmbedThumbnail()
	if thumb == nil {
		thumb = gh.Repo.OwnerThumbnail()
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:       color,
				URL:         gh.PullRequest.HTMLURL,
				Thumbnail:   thumb,
				Author:      gh.Sender.AuthorEmbed(),
				Title:       "Pull request · " + gh.Repo.FullName + " · #" + strconv.Itoa(gh.PullRequest.Number),
				Description: desc,
				Fields: []*discordgo.MessageEmbedField{
					{Name: "Title", Value: gh.PullRequest.Title, Inline: false},
					{Name: "Branches", Value: branches, Inline: false},
					{Name: "PR author", Value: gh.PullRequest.User.Link(), Inline: true},
					{Name: "Actor", Value: gh.Sender.Link(), Inline: true},
					{Name: "Locked", Value: fmt.Sprintf("`%v`", gh.PullRequest.Locked), Inline: true},
				},
			},
		},
	}, nil
}

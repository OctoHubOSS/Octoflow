package events

import (
	"strings"

	"github.com/bwmarrin/discordgo"
)

type CommitCommentEvent struct {
	Action  string     `json:"action"`
	Repo    Repository `json:"repository"`
	Sender  User       `json:"sender"`
	Comment struct {
		Body     string `json:"body"`
		HTMLURL  string `json:"html_url"`
		User     User   `json:"user"`
		CommitID string `json:"commit_id"`
	} `json:"comment"`
}

func commitCommentFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh CommitCommentEvent

	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	comment := strings.TrimSpace(gh.Comment.Body)
	if len(comment) > 1800 {
		comment = comment[:1797] + "…"
	}
	if comment == "" {
		comment = "_Empty comment._"
	}

	color := colorGreen
	if gh.Action == "deleted" {
		color = colorRed
	}

	short := strings.TrimSpace(gh.Comment.CommitID)
	if len(short) > 7 {
		short = short[:7]
	}

	desc := "**Commit:** " + gh.Repo.Commit(gh.Comment.CommitID) + "\n\n" + comment

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:       color,
				URL:         gh.Comment.HTMLURL,
				Thumbnail:   gh.Repo.OwnerThumbnail(),
				Author:      gh.Sender.AuthorEmbed(),
				Title:       "Commit comment · " + gh.Repo.FullName + " · `" + short + "`",
				Description: desc,
				Fields: []*discordgo.MessageEmbedField{
					{Name: "Comment author", Value: gh.Comment.User.Link(), Inline: true},
					{Name: "Actor", Value: gh.Sender.Link(), Inline: true},
				},
			},
		},
	}, nil
}

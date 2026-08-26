package events

import (
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
	}
}

func commitCommentFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh CommitCommentEvent

	// Unmarshal the JSON into our struct
	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
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
				URL:         gh.Comment.HTMLURL,
				Author:      gh.Sender.AuthorEmbed(),
				Title:       "Comment on commit " + gh.Repo.FullName + " (" + gh.Comment.CommitID[:7] + ")",
				Description: comment,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "User",
						Value:  gh.Comment.User.Link(),
						Inline: true,
					},
					{
						Name:   "Commit",
						Value:  gh.Repo.Commit(gh.Comment.CommitID),
						Inline: true,
					},
				},
			},
		},
	}, nil
}

package events

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// BatchedPushFn combines several raw push event bodies (buffered by the
// pneuma batcher over a short window) into one summary embed, instead of
// one message per push. Used only for webhooks with batch_events on - see
// webserver/pneuma/batcher.go.
func BatchedPushFn(bodies [][]byte) (*discordgo.MessageSend, error) {
	var repo Repository
	var sender User
	var commitList string
	totalCommits := 0

	for i, body := range bodies {
		var gh PushEvent

		if err := json.Unmarshal(body, &gh); err != nil {
			continue
		}

		if i == 0 {
			repo = gh.Repo
			sender = gh.Sender
		}

		for _, commit := range gh.Commits {
			totalCommits++

			username := commit.Author.Username
			if username == "" {
				username = commit.Author.Name
			}

			message := truncateText(commit.Message, 100)
			commitList += fmt.Sprintf("%s [`%s`](%s) | %s\n", message, commit.ID[:7], commit.URL, githubUserLink(username))
		}
	}

	commitList = truncateText(commitList, 1024)

	if commitList == "" {
		commitList = "No commits?"
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:     colorGreen,
				Timestamp: nowTimestamp(),
				URL:       repo.HTMLURL,
				Author:    sender.AuthorEmbed(),
				Title:     fmt.Sprintf("%d pushes on %s (%d commits)", len(bodies), repo.FullName, totalCommits),
				Description: "Multiple pushes arrived in quick succession and were combined into this summary. " +
					"Enable this from `/edithook` or the dashboard.",
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:  "Commits",
						Value: commitList,
					},
				},
			},
		},
	}, nil
}

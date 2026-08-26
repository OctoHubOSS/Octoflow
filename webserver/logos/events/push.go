package events

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type PushEvent struct {
	Commits []struct { // push
		ID        string `json:"id"`
		Message   string `json:"message"`
		Timestamp string `json:"timestamp"`
		URL       string `json:"url"`
		Author    struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Username string `json:"username"`
		} `json:"author"`
	} `json:"commits"`
	Repo   Repository `json:"repository"`
	Sender User       `json:"sender"`
	Pusher struct {   // push
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"pusher,omitempty"`
	Ref     string `json:"ref"`
	BaseRef string `json:"base_ref"`
}

func pushFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh PushEvent

	// Unmarshal the JSON into our struct
	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	var commitList string
	for _, commit := range gh.Commits {
		// If the username is empty, use the name instead
		if commit.Author.Username == "" {
			commit.Author.Username = commit.Author.Name
		}

		commit.Message = truncateText(commit.Message, 100)

		commitList += fmt.Sprintf("%s [``%s``](%s) | %s\n", commit.Message, commit.ID[:7], commit.URL, githubUserLink(commit.Author.Username))
	}

	commitList = truncateText(commitList, 1024)

	if commitList == "" {
		commitList = "No commits?"
	}

	branchInfo := "**Ref:** " + gh.Ref

	if gh.BaseRef != "" {
		branchInfo += "\n**Base Ref:** " + gh.BaseRef
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:     colorGreen,
				Timestamp: nowTimestamp(),
				URL:       gh.Repo.HTMLURL,
				Author:    gh.Sender.AuthorEmbed(),
				Title:     "Push on " + gh.Repo.FullName,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:  "Branch",
						Value: branchInfo,
					},
					{
						Name:  "Commits",
						Value: commitList,
					},
					{
						Name:   "Commit Sender",
						Value:  gh.Sender.Link(),
						Inline: true,
					},
					{
						Name:   "Pusher",
						Value:  githubUserLink(gh.Pusher.Name),
						Inline: true,
					},
				},
			},
		},
	}, nil
}

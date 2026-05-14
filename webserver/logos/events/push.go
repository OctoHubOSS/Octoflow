package events

import (
	"fmt"
	"strings"

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

	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	var commitList string
	for _, commit := range gh.Commits {
		if commit.Author.Username == "" {
			commit.Author.Username = commit.Author.Name
		}

		msg := commit.Message
		if len(msg) > 100 {
			msg = msg[:100] + "…"
		}

		hash := strings.TrimSpace(commit.ID)
		if len(hash) >= 7 {
			hash = hash[:7]
		}
		if hash == "" {
			hash = "???????"
		}

		authorURL := "https://github.com/" + strings.ReplaceAll(commit.Author.Username, " ", "%20")
		commitURL := strings.TrimSpace(commit.URL)
		line := fmt.Sprintf("• **%s** · `%s`", msg, hash)
		if commitURL != "" {
			line = fmt.Sprintf("• **%s** [`%s`](%s)", msg, hash, commitURL)
		}
		line += fmt.Sprintf(" · [%s](%s)\n", commit.Author.Username, authorURL)
		commitList += line
	}

	if len(commitList) > 1024 {
		commitList = commitList[:1024] + "…"
	}

	if commitList == "" {
		commitList = "_No commits in payload._"
	}

	n := len(gh.Commits)
	desc := fmt.Sprintf("**%d** commit(s) pushed to **`%s`**", n, gh.Ref)
	if gh.BaseRef != "" {
		desc += "\n**Base ref:** `" + gh.BaseRef + "`"
	}

	pusherVal := gh.Sender.Link()
	if strings.TrimSpace(gh.Pusher.Name) != "" {
		pn := strings.ReplaceAll(gh.Pusher.Name, " ", "%20")
		pusherVal = fmt.Sprintf("[%s](https://github.com/%s)", gh.Pusher.Name, pn)
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:       colorGreen,
				URL:         gh.Repo.HTMLURL,
				Thumbnail:   gh.Repo.OwnerThumbnail(),
				Author:      gh.Sender.AuthorEmbed(),
				Title:       "Push · " + gh.Repo.FullName,
				Description: desc,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:  "Commits",
						Value: commitList,
					},
					{
						Name:   "Pusher (name)",
						Value:  pusherVal,
						Inline: true,
					},
					{
						Name:   "Webhook actor",
						Value:  gh.Sender.Link(),
						Inline: true,
					},
				},
			},
		},
	}, nil
}

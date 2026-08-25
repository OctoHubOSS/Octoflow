package events

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

type CheckSuiteEvent struct {
	Action     string     `json:"action"`
	Repo       Repository `json:"repository"`
	Sender     User       `json:"sender"`
	CheckSuite struct {
		ID         int    `json:"id"`
		After      string `json:"after,omitempty"`
		HeadBranch string `json:"head_branch,omitempty"`
		HeadSHA    string `json:"head_sha,omitempty"`
		Status     string `json:"status,omitempty"`
		Conclusion string `json:"conclusion,omitempty"`
		URL        string `json:"url,omitempty"`
		Before     string `json:"before,omitempty"`
		HeadCommit struct {
			ID        string `json:"id,omitempty"`
			TreeID    string `json:"tree_id,omitempty"`
			Message   string `json:"message,omitempty"`
			Timestamp string `json:"timestamp,omitempty"`
			Author    struct {
				Name     string `json:"name,omitempty"`
				Email    string `json:"email,omitempty"`
				Username string `json:"username,omitempty"`
			} `json:"author,omitempty"`
			Committer struct {
				Name     string `json:"name,omitempty"`
				Email    string `json:"email,omitempty"`
				Username string `json:"username,omitempty"`
			} `json:"committer,omitempty"`
		} `json:"head_commit,omitempty"`
	} `json:"check_suite"`
}

func checkSuiteFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh CheckSuiteEvent

	// Unmarshal the JSON into our struct
	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	if gh.CheckSuite.Conclusion == "" {
		gh.CheckSuite.Conclusion = "No conclusion yet!"
	}

	if gh.CheckSuite.Status == "" {
		gh.CheckSuite.Status = "No status yet!"
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:     colorGreen,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				URL:       gh.Repo.HTMLURL,
				Author:    gh.Sender.AuthorEmbed(),
				Title:     "Check Suite " + gh.Action + " on " + gh.Repo.FullName,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "User",
						Value:  gh.Sender.Link(),
						Inline: true,
					},
					{
						Name:   "Status",
						Value:  gh.CheckSuite.Status,
						Inline: true,
					},
					{
						Name:   "Conclusion",
						Value:  gh.CheckSuite.Conclusion,
						Inline: true,
					},
					{
						Name:   "URL",
						Value:  gh.CheckSuite.URL,
						Inline: true,
					},
					{
						Name:  "Commit",
						Value: gh.CheckSuite.HeadCommit.Message + " | " + gh.Repo.Commit(gh.CheckSuite.HeadCommit.ID),
					},
				},
			},
		},
	}, nil
}

package events

import (
	"strings"

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
		HTMLURL    string `json:"html_url,omitempty"`
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

	page := strings.TrimSpace(gh.CheckSuite.HTMLURL)
	if page == "" {
		page = gh.Repo.HTMLURL
	}

	color := CheckConclusionEmbedColor(gh.CheckSuite.Conclusion)

	msg := strings.TrimSpace(gh.CheckSuite.HeadCommit.Message)
	if len(msg) > 240 {
		msg = msg[:237] + "…"
	}
	if msg == "" {
		msg = "_No commit message._"
	}
	commitLine := msg + "\n" + gh.Repo.Commit(gh.CheckSuite.HeadCommit.ID)

	desc := "**Check suite** `" + gh.Action + "` on " + gh.Repo.MarkdownLink()
	if page != "" && strings.HasPrefix(page, "http") {
		desc += "\n\n[**View check suite**](" + page + ")"
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:       color,
				URL:         page,
				Thumbnail:   gh.Repo.OwnerThumbnail(),
				Author:      gh.Sender.AuthorEmbed(),
				Title:       "Check suite · " + gh.Repo.FullName,
				Description: desc,
				Fields: []*discordgo.MessageEmbedField{
					{Name: "Status", Value: "`" + gh.CheckSuite.Status + "`", Inline: true},
					{Name: "Conclusion", Value: "`" + gh.CheckSuite.Conclusion + "`", Inline: true},
					{Name: "Branch", Value: "`" + gh.CheckSuite.HeadBranch + "`", Inline: true},
					{Name: "Head commit", Value: commitLine, Inline: false},
					{Name: "Actor", Value: gh.Sender.Link(), Inline: true},
				},
			},
		},
	}, nil
}

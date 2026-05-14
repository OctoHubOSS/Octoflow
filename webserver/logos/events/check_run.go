package events

import (
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type CheckRunEvent struct {
	Action   string     `json:"action"`
	Repo     Repository `json:"repository"`
	Sender   User       `json:"sender"`
	CheckRun struct {
		Name       string    `json:"name"`
		HTMLURL    string    `json:"html_url"`
		StartedAt  time.Time `json:"started_at"`
		Status     string    `json:"status"`
		DetailsURL string    `json:"details_url"`
		Conclusion string    `json:"conclusion"`
		HeadSHA    string    `json:"head_sha"`
	} `json:"check_run"`
}

func checkRunFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh CheckRunEvent

	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	if gh.CheckRun.Conclusion == "" {
		gh.CheckRun.Conclusion = "No conclusion yet!"
	}

	if gh.CheckRun.Status == "" {
		gh.CheckRun.Status = "No status yet!"
	}

	page := strings.TrimSpace(gh.CheckRun.HTMLURL)
	if page == "" {
		page = gh.Repo.HTMLURL
	}

	color := CheckConclusionEmbedColor(gh.CheckRun.Conclusion)
	if strings.EqualFold(gh.CheckRun.Status, "in_progress") || strings.EqualFold(gh.CheckRun.Status, "queued") {
		color = colorYellow
	}

	desc := "**" + gh.CheckRun.Name + "** · `" + gh.Action + "` · " + gh.Repo.MarkdownLink()
	if d := strings.TrimSpace(gh.CheckRun.DetailsURL); d != "" {
		desc += "\n\n[**Open details**](" + d + ")"
	}

	ts := ""
	if !gh.CheckRun.StartedAt.IsZero() {
		ts = gh.CheckRun.StartedAt.UTC().Format(time.RFC3339)
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:       color,
				URL:         page,
				Thumbnail:   gh.Repo.OwnerThumbnail(),
				Author:      gh.Sender.AuthorEmbed(),
				Title:       "Check run · " + gh.Repo.FullName,
				Description: desc,
				Timestamp:   ts,
				Fields: []*discordgo.MessageEmbedField{
					{Name: "Status", Value: "`" + gh.CheckRun.Status + "`", Inline: true},
					{Name: "Conclusion", Value: "`" + gh.CheckRun.Conclusion + "`", Inline: true},
					{Name: "SHA", Value: gh.Repo.Commit(gh.CheckRun.HeadSHA), Inline: true},
					{Name: "Actor", Value: gh.Sender.Link(), Inline: false},
				},
			},
		},
	}, nil
}

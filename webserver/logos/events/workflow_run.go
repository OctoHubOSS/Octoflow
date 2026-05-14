package events

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type WorkflowRunEvent struct {
	Action      string     `json:"action"`
	Repo        Repository `json:"repository"`
	Sender      User       `json:"sender"`
	WorkflowRun struct {
		ID              int    `json:"id"`
		HeadBranch      string `json:"head_branch"`
		HeadSHA         string `json:"head_sha"`
		RunNumber       int    `json:"run_number"`
		Event           string `json:"event"`
		Name            string `json:"name"`
		Status          string `json:"status"`
		Conclusion      string `json:"conclusion"`
		URL             string `json:"url"`
		HTMLURL         string `json:"html_url"`
		TriggeringActor User   `json:"triggering_actor"`
		HeadCommit      struct {
			ID        string `json:"id"`
			TreeID    string `json:"tree_id"`
			Message   string `json:"message"`
			Timestamp string `json:"timestamp"`
			Author    User   `json:"author"`
			Committer User   `json:"committer"`
		} `json:"head_commit"`
	} `json:"workflow_run"`
}

func workflowRunFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh WorkflowRunEvent

	// Unmarshal the JSON into our struct
	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	if gh.WorkflowRun.Conclusion == "" {
		gh.WorkflowRun.Conclusion = "No conclusion yet!"
	}

	if gh.WorkflowRun.Status == "" {
		gh.WorkflowRun.Status = "No status yet!"
	}

	page := strings.TrimSpace(gh.WorkflowRun.HTMLURL)
	if page == "" {
		page = gh.Repo.HTMLURL
	}

	color := CheckConclusionEmbedColor(gh.WorkflowRun.Conclusion)
	if strings.EqualFold(gh.WorkflowRun.Status, "in_progress") || strings.EqualFold(gh.WorkflowRun.Status, "queued") || strings.EqualFold(gh.WorkflowRun.Status, "waiting") {
		color = colorYellow
	}

	desc := fmt.Sprintf("**Run #%d** · _%s_", gh.WorkflowRun.RunNumber, gh.Action)
	if msg := strings.TrimSpace(gh.WorkflowRun.HeadCommit.Message); msg != "" {
		if len(msg) > 200 {
			msg = msg[:197] + "…"
		}
		desc += "\n\n**Head commit:** " + msg
	}
	if page != "" && strings.HasPrefix(page, "http") {
		desc += "\n\n[**View workflow run**](" + page + ")"
	}

	trigger := "—"
	if strings.TrimSpace(gh.WorkflowRun.TriggeringActor.Login) != "" {
		trigger = gh.WorkflowRun.TriggeringActor.Link()
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:       color,
				URL:         page,
				Thumbnail:   gh.Repo.OwnerThumbnail(),
				Author:      gh.Sender.AuthorEmbed(),
				Title:       "Actions · " + gh.WorkflowRun.Name,
				Description: desc,
				Fields: []*discordgo.MessageEmbedField{
					{Name: "Status", Value: "`" + gh.WorkflowRun.Status + "`", Inline: true},
					{Name: "Conclusion", Value: "`" + gh.WorkflowRun.Conclusion + "`", Inline: true},
					{Name: "Event", Value: "`" + gh.WorkflowRun.Event + "`", Inline: true},
					{Name: "Branch", Value: "`" + gh.WorkflowRun.HeadBranch + "`", Inline: true},
					{Name: "Commit", Value: gh.Repo.Commit(gh.WorkflowRun.HeadCommit.ID), Inline: true},
					{Name: "Actor", Value: gh.Sender.Link(), Inline: true},
					{Name: "Triggered by", Value: trigger, Inline: false},
				},
			},
		},
	}, nil
}

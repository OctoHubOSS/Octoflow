package events

import (
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type WorkflowJobEvent struct {
	Action      string     `json:"action"`
	Repo        Repository `json:"repository"`
	Sender      User       `json:"sender"`
	WorkflowJob struct {
		ID           int    `json:"id"`
		RunID        int    `json:"run_id"`
		WorkflowName string `json:"workflow_name"`
		HeadBranch   string `json:"head_branch"`
		RunURL       string `json:"run_url"`
		RunAttempt   int    `json:"run_attempt"`
		NodeID       string `json:"node_id"`
		HeadSHA      string `json:"head_sha"`
		URL          string `json:"url"`
		HTMLURL      string `json:"html_url"`
		Status       string `json:"status"`
		Conclusion   string `json:"conclusion"`
		StartedAt    string `json:"started_at"`
		CompletedAt  string `json:"completed_at"`
		Name         string `json:"name"`
		Steps        []struct {
			Name        string `json:"name"`
			Status      string `json:"status"`
			Conclusion  string `json:"conclusion"`
			Number      int    `json:"number"`
			StartedAt   string `json:"started_at"`
			CompletedAt string `json:"completed_at"`
		} `json:"steps"`
		CheckRunUrl     string   `json:"check_run_url"`
		Labels          []string `json:"labels"`
		RunnerID        int      `json:"runner_id"`
		RunnerName      string   `json:"runner_name"`
		RunnerGroupID   int      `json:"runner_group_id"`
		RunnerGroupName string   `json:"runner_group_name"`
	} `json:"workflow_job"`
}

func workflowJobFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh WorkflowJobEvent

	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	if gh.WorkflowJob.Conclusion == "" {
		gh.WorkflowJob.Conclusion = "No conclusion yet!"
	}

	if gh.WorkflowJob.Status == "" {
		gh.WorkflowJob.Status = "No status yet!"
	}

	page := strings.TrimSpace(gh.WorkflowJob.HTMLURL)
	if page == "" {
		page = strings.TrimSpace(gh.WorkflowJob.RunURL)
	}
	if page == "" {
		page = gh.Repo.HTMLURL
	}

	color := CheckConclusionEmbedColor(gh.WorkflowJob.Conclusion)
	if strings.EqualFold(gh.WorkflowJob.Status, "in_progress") || strings.EqualFold(gh.WorkflowJob.Status, "queued") {
		color = colorYellow
	}

	desc := "**" + gh.WorkflowJob.Name + "** · `" + gh.Action + "`\n"
	desc += gh.Repo.MarkdownLink() + "\n**Workflow:** `" + gh.WorkflowJob.WorkflowName + "` · **Attempt:** `" + strconv.Itoa(gh.WorkflowJob.RunAttempt) + "`"
	if page != "" && strings.HasPrefix(page, "http") {
		desc += "\n\n[**View job**](" + page + ")"
	}

	fields := []*discordgo.MessageEmbedField{
		{Name: "Status", Value: "`" + gh.WorkflowJob.Status + "`", Inline: true},
		{Name: "Conclusion", Value: "`" + gh.WorkflowJob.Conclusion + "`", Inline: true},
		{Name: "Branch", Value: "`" + gh.WorkflowJob.HeadBranch + "`", Inline: true},
		{Name: "Commit", Value: gh.Repo.Commit(gh.WorkflowJob.HeadSHA), Inline: true},
		{Name: "Actor", Value: gh.Sender.Link(), Inline: true},
	}

	for _, step := range gh.WorkflowJob.Steps {
		if step.Conclusion == "" {
			step.Conclusion = "No conclusion yet!"
		}
		if step.Status == "" {
			step.Status = "No status yet!"
		}
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Step " + strconv.Itoa(step.Number) + " · " + step.Name,
			Value:  "**Status:** `" + step.Status + "`\n**Conclusion:** `" + step.Conclusion + "`",
			Inline: false,
		})
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:       color,
				URL:         page,
				Thumbnail:   gh.Repo.OwnerThumbnail(),
				Author:      gh.Sender.AuthorEmbed(),
				Title:       "Workflow job · " + gh.Repo.FullName,
				Description: desc,
				Fields:      fields,
			},
		},
	}, nil
}

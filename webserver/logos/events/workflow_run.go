package events

import (
	"fmt"

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

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:     colorGreen,
				Timestamp: nowTimestamp(),
				URL:       gh.Repo.HTMLURL,
				Author:    gh.Sender.AuthorEmbed(),
				Title:     "Workflow Run: " + gh.WorkflowRun.Name,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "User",
						Value:  gh.Sender.Link(),
						Inline: true,
					},
					{
						Name:   "Status",
						Value:  gh.WorkflowRun.Status,
						Inline: true,
					},
					{
						Name:   "Conclusion",
						Value:  gh.WorkflowRun.Conclusion,
						Inline: true,
					},
					{
						Name:   "Branch",
						Value:  gh.WorkflowRun.HeadBranch,
						Inline: true,
					},
					{
						Name:   "Commit",
						Value:  gh.Repo.Commit(gh.WorkflowRun.HeadCommit.ID),
						Inline: true,
					},
					{
						Name:   "URL",
						Value:  gh.WorkflowRun.URL,
						Inline: true,
					},
					{
						Name:   "Event",
						Value:  gh.WorkflowRun.Event,
						Inline: true,
					},
					{
						Name:   "Run Number",
						Value:  fmt.Sprintf("%d", gh.WorkflowRun.RunNumber),
						Inline: true,
					},
					{
						Name:  "Triggered By",
						Value: gh.WorkflowRun.TriggeringActor.Link(),
					},
				},
			},
		},
	}, nil
}

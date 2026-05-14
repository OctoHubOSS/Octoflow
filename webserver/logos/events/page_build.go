package events

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type PageBuildEvent struct {
	Build struct {
		Commit    string    `json:"commit"`
		CreatedAt time.Time `json:"created_at"`
		Duration  int       `json:"duration"`
		Error     struct {
			Message string `json:"message"`
		} `json:"error"`
		Status string `json:"status"`
	} `json:"build"`
	Repo   Repository `json:"repository"`
	Sender User       `json:"sender"`
}

func pageBuildFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh PageBuildEvent

	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	color := colorGreen
	st := strings.ToLower(strings.TrimSpace(gh.Build.Status))
	if st == "errored" || st == "failed" || strings.TrimSpace(gh.Build.Error.Message) != "" {
		color = colorRed
	} else if st == "building" || st == "pending" {
		color = colorYellow
	}

	dur := "unknown"
	if gh.Build.Duration > 0 {
		dur = fmt.Sprintf("%d s", gh.Build.Duration)
	}

	errMsg := strings.TrimSpace(gh.Build.Error.Message)
	if errMsg == "" {
		errMsg = "_None_"
	}

	desc := "**Status:** `" + gh.Build.Status + "` · **Duration:** `" + dur + "`\n**Commit:** " + gh.Repo.Commit(gh.Build.Commit)
	if errMsg != "_None_" {
		desc += "\n\n**Error**\n```\n" + errMsg + "\n```"
	}

	ts := ""
	if !gh.Build.CreatedAt.IsZero() {
		ts = gh.Build.CreatedAt.UTC().Format(time.RFC3339)
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:       color,
				URL:         gh.Repo.HTMLURL,
				Thumbnail:   gh.Repo.OwnerThumbnail(),
				Author:      gh.Sender.AuthorEmbed(),
				Title:       "GitHub Pages · " + gh.Repo.FullName,
				Description: desc,
				Timestamp:   ts,
				Fields: []*discordgo.MessageEmbedField{
					{Name: "Actor", Value: gh.Sender.Link(), Inline: false},
				},
			},
		},
	}, nil
}

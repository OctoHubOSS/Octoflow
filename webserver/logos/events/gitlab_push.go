package events

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type GLPushEvent struct {
	ObjectKind   string       `json:"object_kind"`
	Before       string       `json:"before"`
	After        string       `json:"after"`
	Ref          string       `json:"ref"`
	CheckoutSHA  string       `json:"checkout_sha"`
	UserID       int          `json:"user_id"`
	UserName     string       `json:"user_name"`
	UserUsername string       `json:"user_username"`
	UserEmail    string       `json:"user_email"`
	UserAvatar   string       `json:"user_avatar"`
	Project      GLProject    `json:"project"`
	Repository   GLRepository `json:"repository"`
	Commits      []struct {
		ID        string `json:"id"`
		Message   string `json:"message"`
		Title     string `json:"title"`
		Timestamp string `json:"timestamp"`
		URL       string `json:"url"`
		Author    struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"author"`
	} `json:"commits"`
	TotalCommitsCount int `json:"total_commits_count"`
}

func glPushFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gl GLPushEvent
	err := json.Unmarshal(bytes, &gl)
	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	var commitList string
	for _, commit := range gl.Commits {
		msg := commit.Message
		if len(msg) > 100 {
			msg = msg[:100] + "..."
		}
		commitList += fmt.Sprintf("• **%s** [`%s`](%s) · _%s_\n", msg, commit.ID[:7], commit.URL, commit.Author.Name)
	}

	if len(commitList) > 1024 {
		commitList = commitList[:1024] + "..."
	}

	if commitList == "" {
		commitList = "No commits"
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:     glColorOrange,
				URL:       gl.Project.WebURL,
				Thumbnail: glProjectThumbnail(gl.Project),
				Author: &discordgo.MessageEmbedAuthor{
					Name:    gl.UserName + " (@" + gl.UserUsername + ")",
					IconURL: gl.UserAvatar,
				},
				Title:       "Push · " + gl.Project.PathWithNamespace,
				Description: fmt.Sprintf("**%d** commit(s) on `%s`", gl.TotalCommitsCount, gl.Ref),
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:  "Commits",
						Value: commitList,
					},
					{
						Name:   "Pusher",
						Value:  fmt.Sprintf("[%s](%s/%s)", gl.UserUsername, strings.TrimSuffix(gl.Project.WebURL, "/"+gl.Project.PathWithNamespace), gl.UserUsername),
						Inline: true,
					},
					{
						Name:   "Total",
						Value:  fmt.Sprintf("%d", gl.TotalCommitsCount),
						Inline: true,
					},
				},
				Footer: glFooterForProject(gl.Project),
			},
		},
	}, nil
}

type GLTagPushEvent struct {
	ObjectKind   string       `json:"object_kind"`
	Before       string       `json:"before"`
	After        string       `json:"after"`
	Ref          string       `json:"ref"`
	CheckoutSHA  string       `json:"checkout_sha"`
	UserID       int          `json:"user_id"`
	UserName     string       `json:"user_name"`
	UserUsername string       `json:"user_username"`
	UserAvatar   string       `json:"user_avatar"`
	Project      GLProject    `json:"project"`
	Repository   GLRepository `json:"repository"`
}

func glTagPushFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gl GLTagPushEvent
	err := json.Unmarshal(bytes, &gl)
	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:     glColorOrange,
				URL:       gl.Project.WebURL,
				Thumbnail: glProjectThumbnail(gl.Project),
				Author: &discordgo.MessageEmbedAuthor{
					Name:    gl.UserName + " (@" + gl.UserUsername + ")",
					IconURL: gl.UserAvatar,
				},
				Title:       "Tag push · " + gl.Project.PathWithNamespace,
				Description: "New tag activity on **`" + gl.Ref + "`**.",
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:  "Ref",
						Value: "`" + gl.Ref + "`",
					},
					{
						Name:   "Checkout SHA",
						Value:  "`" + gl.CheckoutSHA + "`",
						Inline: true,
					},
				},
				Footer: glFooterForProject(gl.Project),
			},
		},
	}, nil
}

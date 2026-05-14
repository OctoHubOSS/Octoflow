package events

import (
	"strings"

	"github.com/bwmarrin/discordgo"
)

type StatusEvent struct {
	Repo        Repository `json:"repository"`
	Sender      User       `json:"sender"`
	State       string     `json:"state"`
	Description string     `json:"description"`
	TargetURL   string     `json:"target_url"`
	Context     string     `json:"context"`
	Commit      struct {   // status
		HTMLURL string `json:"html_url"`
		SHA     string `json:"sha"`
		Author  struct {
			Login   string `json:"login"`
			HTMLURL string `json:"html_url"` // user
		} `json:"author"`
		Commit struct {
			ID      string `json:"id"`
			Message string `json:"message"`
		} `json:"commit"`
	} `json:"commit"`
}

func statusFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh StatusEvent

	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	color := colorYellow
	switch strings.ToLower(strings.TrimSpace(gh.State)) {
	case "success", "successful":
		color = colorGreen
	case "failure", "error", "failed":
		color = colorRed
	case "pending":
		color = colorYellow
	}

	ctx := strings.TrimSpace(gh.Context)
	if ctx == "" {
		ctx = "—"
	}

	desc := strings.TrimSpace(gh.Description)
	if desc == "" {
		desc = "_No status description._"
	}
	if u := strings.TrimSpace(gh.TargetURL); u != "" {
		desc += "\n\n[**Open status target**](" + u + ")"
	}

	sha := strings.TrimSpace(gh.Commit.SHA)
	commitLine := gh.Repo.Commit(sha)
	msg := strings.TrimSpace(gh.Commit.Commit.Message)
	if len(msg) > 200 {
		msg = msg[:197] + "…"
	}
	if msg != "" {
		commitLine += "\n_" + msg + "_"
	}

	author := User{Login: gh.Commit.Author.Login, HTMLURL: gh.Commit.Author.HTMLURL}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:       color,
				URL:         gh.Repo.HTMLURL,
				Thumbnail:   gh.Repo.OwnerThumbnail(),
				Author:      gh.Sender.AuthorEmbed(),
				Title:       "Commit status · " + gh.Repo.FullName + " · `" + gh.State + "`",
				Description: desc,
				Fields: []*discordgo.MessageEmbedField{
					{Name: "Commit", Value: commitLine, Inline: false},
					{Name: "Context", Value: "`" + ctx + "`", Inline: true},
					{Name: "Commit author", Value: author.Link(), Inline: true},
					{Name: "Webhook actor", Value: gh.Sender.Link(), Inline: true},
				},
			},
		},
	}, nil
}

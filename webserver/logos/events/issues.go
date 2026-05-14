package events

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type IssuesEvent struct {
	Action string     `json:"action"`
	Repo   Repository `json:"repository"`
	Sender User       `json:"sender"`
	Issue  Issue      `json:"issue"`
}

func issuesFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh IssuesEvent

	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	var body string = gh.Issue.Body
	if len(gh.Issue.Body) > 996 {
		body = gh.Issue.Body[:996] + "…"
	}

	if body == "" {
		body = "_No description provided._"
	}

	actionLabel := cases.Title(language.English).String(strings.ReplaceAll(gh.Action, "_", " "))

	color := colorGreen
	switch gh.Action {
	case "deleted", "unpinned", "demilestoned", "closed", "locked":
		color = colorRed
	case "edited", "labeled", "unlabeled", "assigned", "unassigned", "milestoned", "pinned":
		color = colorYellow
	}

	desc := fmt.Sprintf("**%s** · #%d · **`%s`**\n\n%s", actionLabel, gh.Issue.Number, gh.Issue.State, body)

	thumb := gh.Issue.User.EmbedThumbnail()
	if thumb == nil {
		thumb = gh.Repo.OwnerThumbnail()
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:       color,
				URL:         gh.Issue.HTMLURL,
				Thumbnail:   thumb,
				Author:      gh.Sender.AuthorEmbed(),
				Description: desc,
				Title:       "Issue · " + gh.Repo.FullName + " · #" + strconv.Itoa(gh.Issue.Number),
				Fields: []*discordgo.MessageEmbedField{
					{Name: "Title", Value: gh.Issue.Title, Inline: false},
					{Name: "Author", Value: gh.Issue.User.Link(), Inline: true},
					{Name: "Actor", Value: gh.Sender.Link(), Inline: true},
				},
			},
		},
	}, nil
}

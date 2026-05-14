package events

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type CreateEvent struct {
	Repo         Repository `json:"repository"`
	Sender       User       `json:"sender"`
	Ref          string     `json:"ref"`
	RefType      string     `json:"ref_type"`
	MasterBranch string     `json:"master_branch"`
	PusherType   string     `json:"pusher_type"`
}

func createFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh CreateEvent

	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	desc := fmt.Sprintf("Created **`%s`** (`%s`)", gh.RefType, gh.Ref)
	if gh.MasterBranch != "" {
		desc += "\n**Default branch:** `" + gh.MasterBranch + "`"
	}
	if gh.PusherType != "" {
		desc += "\n**Pusher type:** `" + gh.PusherType + "`"
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:       colorGreen,
				URL:         gh.Repo.HTMLURL,
				Thumbnail:   gh.Repo.OwnerThumbnail(),
				Author:      gh.Sender.AuthorEmbed(),
				Title:       "Create · " + gh.Repo.FullName,
				Description: desc,
				Fields: []*discordgo.MessageEmbedField{
					{Name: "Actor", Value: gh.Sender.Link(), Inline: false},
				},
			},
		},
	}, nil
}

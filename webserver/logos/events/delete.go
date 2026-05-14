package events

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type DeleteEvent struct {
	Repo       Repository `json:"repository"`
	Sender     User       `json:"sender"`
	Ref        string     `json:"ref"`
	RefType    string     `json:"ref_type"`
	PusherType string     `json:"pusher_type"`
}

func deleteFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh DeleteEvent

	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	desc := fmt.Sprintf("Deleted **`%s`** (`%s`)", gh.RefType, gh.Ref)
	if gh.PusherType != "" {
		desc += "\n**Pusher type:** `" + gh.PusherType + "`"
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:       colorRed,
				URL:         gh.Repo.HTMLURL,
				Thumbnail:   gh.Repo.OwnerThumbnail(),
				Author:      gh.Sender.AuthorEmbed(),
				Title:       "Delete · " + gh.Repo.FullName,
				Description: desc,
				Fields: []*discordgo.MessageEmbedField{
					{Name: "Actor", Value: gh.Sender.Link(), Inline: false},
				},
			},
		},
	}, nil
}

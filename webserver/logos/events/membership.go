package events

import (
	"github.com/bwmarrin/discordgo"
)

type MembershipEvent struct {
	Action string `json:"action"`
	Scope  string `json:"scope"`
	Sender User   `json:"sender"`
	Member User   `json:"member"`
	Team   struct {
		Name    string `json:"name"`
		HTMLUrl string `json:"html_url"`
	} `json:"team"`
	Organization struct {
		Login string `json:"login"`
	} `json:"organization"`
}

func membershipFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh MembershipEvent

	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	var color int
	if gh.Action == "removed" {
		color = colorRed
	} else {
		color = colorGreen
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:     color,
				Timestamp: nowTimestamp(),
				URL:       gh.Team.HTMLUrl,
				Author:    gh.Sender.AuthorEmbed(),
				Title:     "Team membership " + gh.Action + " in " + gh.Organization.Login,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Member",
						Value:  gh.Member.Link(),
						Inline: true,
					},
					{
						Name:   "Team",
						Value:  gh.Team.Name,
						Inline: true,
					},
					{
						Name:   "By",
						Value:  gh.Sender.Link(),
						Inline: true,
					},
				},
			},
		},
	}, nil
}

package events

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

type DiscussionCommentEvent struct {
	Action  string     `json:"action"`     // The type of action for the event
	Repo    Repository `json:"repository"` // The repository where this event was created
	Sender  User       `json:"sender"`     // The user who sent/created the event
	Comment struct {
		Author  User      `json:"user"`       // Author of the comment
		Content string    `json:"body"`       // The content of the comment
		Created time.Time `json:"created_at"` // Comment creation
		Url     string    `json:"html_url"`   // Url to the comment
	} `json:"comment"`
	Discussion struct {
		Title    string `json:"title"` // The title of the origin discussion
		Category struct {
			Name        string `json:"name"`          // Name of the discussion category
			Answerable  bool   `json:"is_answerable"` // If the discussion can have a valid answer
			Description string `json:"description"`   // The description of the discussion category
		} `json:"category"`
		Date time.Time `json:"created_at"` // Date the discussion was created
		Url  string    `json:"html_url"`   // URL to the discussion
	} `json:"discussion"`
}

func discussionCommentFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh DiscussionCommentEvent

	// Unmarshall the json into our struct
	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	switch gh.Action {

	case "created":

		if len(gh.Comment.Content) > 3000 {
			gh.Comment.Content = gh.Comment.Content[:3000] + "... [Read More](" + gh.Comment.Url + ")"
		}

		if len(gh.Discussion.Title) > 190 {
			gh.Discussion.Title = gh.Discussion.Title[:190] + "... [View Discussion](" + gh.Discussion.Url + ")"
		}

		return &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{
				{
					Color:       colorGreen,
					URL:         gh.Repo.HTMLURL,
					Title:       "New Comment on Discussion",
					Author:      gh.Sender.AuthorEmbed(),
					Description: gh.Comment.Content,
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:   "Discussion",
							Value:  gh.Discussion.Title,
							Inline: true,
						},
						{
							Name:   "Comment Author",
							Value:  gh.Sender.Link(),
							Inline: true,
						},
						{
							Name:   "Repository",
							Value:  gh.Repo.MarkdownLink(),
							Inline: true,
						},
					},
					Timestamp: gh.Comment.Created.Format(time.RFC3339),
				},
			},
		}, nil

	case "edited":

		if len(gh.Comment.Content) > 3000 {
			gh.Comment.Content = gh.Comment.Content[:3000] + "... [Read More](" + gh.Comment.Url + ")"
		}

		if len(gh.Discussion.Title) > 190 {
			gh.Discussion.Title = gh.Discussion.Title[:190] + "... [View Discussion](" + gh.Discussion.Url + ")"
		}

		return &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{
				{
					Color:       colorGreen,
					URL:         gh.Repo.HTMLURL,
					Title:       "Discussion Comment Updated",
					Author:      gh.Sender.AuthorEmbed(),
					Description: gh.Comment.Content,
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:   "Discussion",
							Value:  gh.Discussion.Title,
							Inline: true,
						},
						{
							Name:   "Comment Author",
							Value:  gh.Comment.Author.Link(),
							Inline: true,
						},
						{
							Name:   "Repository",
							Value:  gh.Repo.MarkdownLink(),
							Inline: true,
						},
					},
					Timestamp: gh.Comment.Created.Format(time.RFC3339),
				},
			},
		}, nil

	case "deleted":

		if len(gh.Comment.Content) > 3000 {
			gh.Comment.Content = gh.Comment.Content[:3000] + "... [Read More](" + gh.Comment.Url + ")"
		}

		if len(gh.Discussion.Title) > 200 {
			gh.Discussion.Title = gh.Discussion.Title[:200] + "... [View Discussion](" + gh.Discussion.Url + ")"
		}

		return &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{
				{
					Color:       colorGreen,
					URL:         gh.Repo.HTMLURL,
					Title:       "Comment Deleted",
					Author:      gh.Sender.AuthorEmbed(),
					Description: gh.Comment.Content,
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:   "Discussion",
							Value:  gh.Discussion.Title,
							Inline: true,
						},
						{
							Name:   "Comment Author",
							Value:  gh.Comment.Author.Link(),
							Inline: true,
						},
						{
							Name:   "Repository",
							Value:  gh.Repo.MarkdownLink(),
							Inline: true,
						},
						{
							Name:   "Deleted By",
							Value:  gh.Sender.Link(),
							Inline: true,
						},
					},
					Timestamp: gh.Comment.Created.Format(time.RFC3339),
				},
			},
		}, nil

	default:

		if len(gh.Discussion.Title) > 200 {
			gh.Discussion.Title = gh.Discussion.Title[:200] + "... [View Discussion](" + gh.Discussion.Url + ")"
		}

		return &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{
				{
					URL:         gh.Repo.HTMLURL,
					Color:       colorRed,
					Author:      gh.Sender.AuthorEmbed(),
					Title:       "Discussion Comment Updated",
					Description: "It looks like this comment has received a update that is not tracked by our systems yet!",
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:   "Repository",
							Value:  gh.Repo.MarkdownLink(),
							Inline: true,
						},
						{
							Name:   "Discussion",
							Value:  gh.Discussion.Title,
							Inline: true,
						},
						{
							Name:   "Comment Author",
							Value:  gh.Comment.Author.Link(),
							Inline: true,
						},
					},
					Timestamp: gh.Comment.Created.Format(time.RFC3339),
				},
			},
		}, nil
	}
}

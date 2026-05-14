package events

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

type DiscussionEvent struct {
	Action string     `json:"action"`     // The type of action for the event
	Repo   Repository `json:"repository"` // The repository where this event was created
	Sender User       `json:"sender"`     // The user who sent/created the event
	Label  struct {
		Name        string `json:"name"`        // Name of the discussion label
		Default     bool   `json:"default"`     // Is the label our a default label
		Description string `json:"description"` // Description of the label
	} `json:"label"`
	Discussion struct {
		Title            string `json:"title"`              // The title of the origin discussion
		Author           User   `json:"user"`               // The user who originally posted the discussion/answer
		Created          string `json:"answer_chosen_at"`   // The date the answer was chosen at
		AnswerSubmitter  User   `json:"answer_chosen_by"`   // The user who marked the answer as correct
		ActiveLockReason string `json:"active_lock_reason"` // Reason for the discussion being locked (no comments allowed)
		AnswerHtmlUrl    string `json:"answer_html_url"`    // URL To the answer/comment
		AnswerRespBody   string `json:"body"`               // Body/content of the answer
		Category         struct {
			Name        string `json:"name"`          // Name of the discussion category
			Answerable  bool   `json:"is_answerable"` // If the discussion can have a valid answer
			Description string `json:"description"`   // The description of the discussion category
		} `json:"category"`
		DiscussionDate time.Time `json:"created_at"` // Date the discussion was created
		DiscussionURL  string    `json:"html_url"`   // URL to the discussion
	} `json:"discussion"`
}

func discussionFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh DiscussionEvent

	// Unmarshall the json into our struct
	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	switch gh.Action {

	// Someone posted a suggested answer/fix
	case "answered":

		var title string = "Discussion Answered"

		var locked = gh.Discussion.ActiveLockReason

		if gh.Discussion.ActiveLockReason != "" {
			locked = "Discussion is now Closed: " + gh.Discussion.ActiveLockReason
		} else {
			locked = "Discussion is still open for comments"
		}

		if len(gh.Discussion.AnswerRespBody) > 3000 {
			gh.Discussion.AnswerRespBody = gh.Discussion.AnswerRespBody[:3000] + "... [View Discussion](" + gh.Discussion.DiscussionURL + ")"
		}

		if len(gh.Discussion.Title) > 190 {
			gh.Discussion.Title = gh.Discussion.Title[:190] + "... [View Discussion](" + gh.Discussion.DiscussionURL + ")"
		}

		return &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{
				{
					Color:       colorGreen,
					URL:         gh.Repo.HTMLURL,
					Title:       title,
					Author:      gh.Sender.AuthorEmbed(),
					Description: gh.Discussion.AnswerRespBody,
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:   "Repository",
							Value:  gh.Repo.MarkdownLink(),
							Inline: true,
						},
						{
							Name:   "Answer Selected By",
							Value:  gh.Discussion.AnswerSubmitter.Link(),
							Inline: true,
						},
						{
							Name:   "Answer Posted By",
							Value:  gh.Discussion.Author.Link(),
							Inline: true,
						},
						{
							Name:   "Origin Discussion",
							Value:  gh.Discussion.Title,
							Inline: true,
						},
						{
							Name:   "View Answer",
							Value:  gh.Discussion.AnswerHtmlUrl,
							Inline: true,
						},
						{
							Name:   "View Discussion",
							Value:  gh.Discussion.DiscussionURL,
							Inline: true,
						},
						{
							Name:   "Discussion Locked?",
							Value:  locked,
							Inline: true,
						},
					},
					Timestamp: gh.Discussion.DiscussionDate.Format(time.RFC3339),
				},
			},
		}, nil

	// Someone changed the category of the discussion
	case "category_changed":

		if len(gh.Discussion.Title) > 190 {
			gh.Discussion.Title = gh.Discussion.Title[:190] + "... [View Discussion](" + gh.Discussion.DiscussionURL + ")"
		}

		return &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{
				{
					Color:       colorYellow,
					URL:         gh.Repo.HTMLURL,
					Title:       "Discussion Category Updated",
					Author:      gh.Sender.AuthorEmbed(),
					Description: "This discussion has been moved to a new category!",
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
							Name:   "New Category",
							Value:  gh.Discussion.Category.Name,
							Inline: true,
						},
					},
					Timestamp: gh.Discussion.DiscussionDate.Format(time.RFC3339),
				},
			},
		}, nil

	// Someone closed the discussion and comments are no longer allowed
	case "closed":

		if len(gh.Discussion.Title) > 190 {
			gh.Discussion.Title = gh.Discussion.Title[:190] + "..."
		}

		return &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{
				{
					Color:       colorRed,
					URL:         gh.Repo.HTMLURL,
					Title:       "Discussion Closed",
					Author:      gh.Sender.AuthorEmbed(),
					Description: "This discussion has been closed and will no longer allow new comments/posts",
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:   "Discussion",
							Value:  gh.Discussion.Title,
							Inline: true,
						},
						{
							Name:   "Closed By",
							Value:  gh.Sender.Link(),
							Inline: true,
						},
						{
							Name:   "Created By",
							Value:  gh.Discussion.Author.Link(),
							Inline: true,
						},
						{
							Name:   "Repository",
							Value:  gh.Repo.MarkdownLink(),
							Inline: true,
						},
					},
					Timestamp: gh.Discussion.DiscussionDate.Format(time.RFC3339),
				},
			},
		}, nil

	// Discussion has been reopened
	case "reopened":

		if len(gh.Discussion.Title) > 190 {
			gh.Discussion.Title = gh.Discussion.Title[:190] + "... [View Discussion](" + gh.Discussion.DiscussionURL + ")"
		}

		return &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{
				{
					Color:       colorRed,
					URL:         gh.Repo.HTMLURL,
					Title:       "Discussion Reopened",
					Author:      gh.Sender.AuthorEmbed(),
					Description: "This discussion has been reopened",
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:   "Discussion",
							Value:  gh.Discussion.Title,
							Inline: true,
						},
						{
							Name:   "Opened By",
							Value:  gh.Sender.Link(),
							Inline: true,
						},
						{
							Name:   "Created By",
							Value:  gh.Discussion.Author.Link(),
							Inline: true,
						},
						{
							Name:   "Repository",
							Value:  gh.Repo.MarkdownLink(),
							Inline: true,
						},
					},
					Timestamp: gh.Discussion.DiscussionDate.Format(time.RFC3339),
				},
			},
		}, nil

	// Someone created a new discussion
	case "created":

		if len(gh.Discussion.Title) > 190 {
			gh.Discussion.Title = gh.Discussion.Title[:190] + "... [View Discussion](" + gh.Discussion.DiscussionURL + ")"
		}

		if len(gh.Discussion.AnswerRespBody) > 3000 {
			gh.Discussion.AnswerRespBody = gh.Discussion.AnswerRespBody[:3000] + "... [View Discussion](" + gh.Discussion.DiscussionURL + ")"
		}

		return &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{
				{
					URL:         gh.Repo.HTMLURL,
					Color:       colorGreen,
					Title:       "New Discussion Created",
					Author:      gh.Sender.AuthorEmbed(),
					Description: gh.Discussion.AnswerRespBody,
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:   "Title",
							Value:  gh.Discussion.Title,
							Inline: true,
						},
						{
							Name:   "Author",
							Value:  gh.Sender.Link(),
							Inline: true,
						},
						{
							Name:   "Category",
							Value:  gh.Discussion.Category.Name,
							Inline: true,
						},
						{
							Name:   "Repository",
							Value:  gh.Repo.MarkdownLink(),
							Inline: true,
						},
					},
					Timestamp: gh.Discussion.DiscussionDate.Format(time.RFC3339),
				},
			},
		}, nil

	// Someone deleted a discussion
	case "deleted":

		var descriptionString string = gh.Discussion.Title + " has been deleted by: " + gh.Sender.AuthorEmbed().Name

		return &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{
				{
					URL:         gh.Repo.HTMLURL,
					Color:       colorRed,
					Author:      gh.Sender.AuthorEmbed(),
					Title:       "Discussion Deleted",
					Description: descriptionString,
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:   "Repository",
							Value:  gh.Repo.MarkdownLink(),
							Inline: true,
						},
					},
					Timestamp: gh.Discussion.DiscussionDate.Format(time.RFC3339),
				},
			},
		}, nil

	// Discussion has been edited
	case "edited":

		if len(gh.Discussion.AnswerRespBody) > 3000 {
			gh.Discussion.AnswerRespBody = gh.Discussion.AnswerRespBody[:3000] + "... [View Discussion](" + gh.Discussion.DiscussionURL + ")"
		}

		if len(gh.Discussion.Title) > 190 {
			gh.Discussion.Title = gh.Discussion.Title[:190] + "... [View Discussion](" + gh.Discussion.DiscussionURL + ")"
		}

		return &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{
				{
					URL:         gh.Repo.HTMLURL,
					Color:       colorYellow,
					Author:      gh.Sender.AuthorEmbed(),
					Title:       "Discussion Updated",
					Description: gh.Discussion.AnswerRespBody,
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
					},
					Timestamp: gh.Discussion.DiscussionDate.Format(time.RFC3339),
				},
			},
		}, nil

	// Discussion has been labeled
	case "labeled":

		var defaultString string

		if gh.Label.Default {
			defaultString = "Yes"
		} else {
			defaultString = "No"
		}

		if len(gh.Discussion.Title) > 190 {
			gh.Discussion.Title = gh.Discussion.Title[:190] + "... [View Discussion](" + gh.Discussion.DiscussionURL + ")"
		}

		return &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{
				{
					URL:    gh.Repo.HTMLURL,
					Color:  colorGreen,
					Author: gh.Sender.AuthorEmbed(),
					Title:  "Discussion Label Added",
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:   "Discussion",
							Value:  gh.Discussion.Title,
							Inline: true,
						},
						{
							Name:   "Category",
							Value:  gh.Discussion.Category.Name,
							Inline: true,
						},
						{
							Name:   "Label Name",
							Value:  gh.Label.Name,
							Inline: true,
						},
						{
							Name:   "Label Description",
							Value:  gh.Label.Description,
							Inline: true,
						},
						{
							Name:   "Default Label?",
							Value:  defaultString,
							Inline: true,
						},
						{
							Name:   "Added By",
							Value:  gh.Sender.Link(),
							Inline: true,
						},
						{
							Name:   "Repository",
							Value:  gh.Repo.MarkdownLink(),
							Inline: true,
						},
					},
					Timestamp: gh.Discussion.DiscussionDate.Format(time.RFC3339),
				},
			},
		}, nil

	// Discussion has been locked by moderator
	case "locked":

		var lockedReason string

		if gh.Discussion.ActiveLockReason == "" {
			lockedReason = "No reason provided"
		} else if len(gh.Discussion.ActiveLockReason) > 999 {
			lockedReason = gh.Discussion.AnswerRespBody[:999] + "... [View Discussion](" + gh.Discussion.DiscussionURL + ")"
		} else {
			lockedReason = gh.Discussion.ActiveLockReason
		}

		if len(gh.Discussion.Title) > 190 {
			gh.Discussion.Title = gh.Discussion.Title[:190] + "... [View Discussion](" + gh.Discussion.DiscussionURL + ")"
		}

		return &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{
				{
					URL:         gh.Repo.HTMLURL,
					Color:       colorRed,
					Author:      gh.Sender.AuthorEmbed(),
					Title:       "Discussion Locked",
					Description: "Adding new comments/answers is now prohibited",
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:   "Discussion",
							Value:  gh.Discussion.Title,
							Inline: true,
						},
						{
							Name:   "Reason",
							Value:  lockedReason,
							Inline: true,
						},
						{
							Name:   "Locked By",
							Value:  gh.Sender.Link(),
							Inline: true,
						},
						{
							Name:   "Repository",
							Value:  gh.Repo.MarkdownLink(),
							Inline: true,
						},
					},
					Timestamp: gh.Discussion.DiscussionDate.Format(time.RFC3339),
				},
			},
		}, nil

	// Discussion has been unlocked
	case "unlocked":

		if len(gh.Discussion.Title) > 190 {
			gh.Discussion.Title = gh.Discussion.Title[:190] + "... [View Discussion](" + gh.Discussion.DiscussionURL + ")"
		}

		return &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{
				{
					URL:         gh.Repo.HTMLURL,
					Color:       colorRed,
					Author:      gh.Sender.AuthorEmbed(),
					Title:       "Discussion UnLocked",
					Description: "You are free to answer/comment again",
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:   "Discussion",
							Value:  gh.Discussion.Title,
							Inline: true,
						},
						{
							Name:   "Unlocked By",
							Value:  gh.Sender.Link(),
							Inline: true,
						},
						{
							Name:   "Repository",
							Value:  gh.Repo.MarkdownLink(),
							Inline: true,
						},
					},
					Timestamp: gh.Discussion.DiscussionDate.Format(time.RFC3339),
				},
			},
		}, nil

	// Discussion has been pinned
	case "pinned":

		if len(gh.Discussion.AnswerRespBody) > 3000 {
			gh.Discussion.AnswerRespBody = gh.Discussion.AnswerRespBody[:3000] + "... [View Discussion](" + gh.Discussion.DiscussionURL + ")"
		}

		if len(gh.Discussion.Title) > 190 {
			gh.Discussion.Title = gh.Discussion.Title[:190] + "... [View Discussion](" + gh.Discussion.DiscussionURL + ")"
		}

		return &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{
				{
					URL:         gh.Repo.HTMLURL,
					Color:       colorGreen,
					Author:      gh.Sender.AuthorEmbed(),
					Title:       "Discussion Pinned",
					Description: gh.Discussion.AnswerRespBody,
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:   "Discussion",
							Value:  gh.Discussion.Title,
							Inline: true,
						},
						{
							Name:   "Category",
							Value:  gh.Discussion.Category.Name,
							Inline: true,
						},
						{
							Name:   "Author",
							Value:  gh.Discussion.Author.Link(),
							Inline: true,
						},
						{
							Name:   "Pinned By",
							Value:  gh.Sender.Link(),
							Inline: true,
						},
					},
					Timestamp: gh.Discussion.DiscussionDate.Format(time.RFC3339),
				},
			},
		}, nil

	// Case has been unpinned
	case "unpinned":

		if len(gh.Discussion.AnswerRespBody) > 3000 {
			gh.Discussion.AnswerRespBody = gh.Discussion.AnswerRespBody[:3000] + "... [View Discussion](" + gh.Discussion.DiscussionURL + ")"
		}

		if len(gh.Discussion.Title) > 190 {
			gh.Discussion.Title = gh.Discussion.Title[:190] + "... [View Discussion](" + gh.Discussion.DiscussionURL + ")"
		}

		return &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{
				{
					URL:         gh.Repo.HTMLURL,
					Color:       colorGreen,
					Author:      gh.Sender.AuthorEmbed(),
					Title:       "Discussion UnPinned",
					Description: gh.Discussion.AnswerRespBody,
					Fields: []*discordgo.MessageEmbedField{
						{
							Name:   "Discussion",
							Value:  gh.Discussion.Title,
							Inline: true,
						},
						{
							Name:   "Category",
							Value:  gh.Discussion.Category.Name,
							Inline: true,
						},
						{
							Name:   "Author",
							Value:  gh.Discussion.Author.Link(),
							Inline: true,
						},
						{
							Name:   "Unpinned By",
							Value:  gh.Sender.Link(),
							Inline: true,
						},
					},
					Timestamp: gh.Discussion.DiscussionDate.Format(time.RFC3339),
				},
			},
		}, nil

	// Default response if we do not track the requested discussion event
	default:

		if len(gh.Discussion.Title) > 190 {
			gh.Discussion.Title = gh.Discussion.Title[:190] + "... [View Discussion](" + gh.Discussion.DiscussionURL + ")"
		}

		return &discordgo.MessageSend{
			Embeds: []*discordgo.MessageEmbed{
				{
					URL:         gh.Repo.HTMLURL,
					Color:       colorRed,
					Author:      gh.Sender.AuthorEmbed(),
					Title:       "Discussion Updated",
					Description: "It looks like this discussion has received a update that is not tracked by our systems yet!",
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
							Name:   "Action Taken",
							Value:  gh.Action,
							Inline: true,
						},
					},
					Timestamp: gh.Discussion.DiscussionDate.Format(time.RFC3339),
				},
			},
		}, nil
	}
}

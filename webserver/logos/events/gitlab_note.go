package events

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type GLNoteEvent struct {
	ObjectKind       string    `json:"object_kind"`
	EventType        string    `json:"event_type"`
	User             GLUser    `json:"user"`
	Project          GLProject `json:"project"`
	ObjectAttributes struct {
		ID           int    `json:"id"`
		Note         string `json:"note"`
		NoteableType string `json:"noteable_type"`
		URL          string `json:"url"`
	} `json:"object_attributes"`
	Issue *struct {
		IID   int    `json:"iid"`
		Title string `json:"title"`
	} `json:"issue,omitempty"`
	MergeRequest *struct {
		IID   int    `json:"iid"`
		Title string `json:"title"`
	} `json:"merge_request,omitempty"`
	Commit *struct {
		ID      string `json:"id"`
		Message string `json:"message"`
	} `json:"commit,omitempty"`
	Snippet *struct {
		ID    int    `json:"id"`
		Title string `json:"title"`
	} `json:"snippet,omitempty"`
}

func glNoteFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gl GLNoteEvent
	err := json.Unmarshal(bytes, &gl)
	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	note := gl.ObjectAttributes.Note
	if len(note) > 1800 {
		note = note[:1797] + "…"
	}

	var target string
	switch gl.ObjectAttributes.NoteableType {
	case "Issue":
		if gl.Issue != nil {
			target = fmt.Sprintf("Issue #%d · %s", gl.Issue.IID, gl.Issue.Title)
		} else {
			target = "Issue"
		}
	case "MergeRequest":
		if gl.MergeRequest != nil {
			target = fmt.Sprintf("Merge request !%d · %s", gl.MergeRequest.IID, gl.MergeRequest.Title)
		} else {
			target = "Merge request"
		}
	case "Commit":
		if gl.Commit != nil {
			shortID := gl.Commit.ID
			if len(shortID) > 7 {
				shortID = shortID[:7]
			}
			target = "Commit `" + shortID + "`"
		} else {
			target = "Commit"
		}
	case "Snippet":
		if gl.Snippet != nil {
			target = fmt.Sprintf("Snippet #%d · %s", gl.Snippet.ID, gl.Snippet.Title)
		} else {
			target = "Snippet"
		}
	default:
		target = gl.ObjectAttributes.NoteableType
	}

	desc := "_On **" + target + "**_\n\n" + note

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:       glColorPurple,
				URL:         gl.ObjectAttributes.URL,
				Thumbnail:   glProjectThumbnail(gl.Project),
				Author:      gl.User.AuthorEmbed(),
				Title:       "Comment · " + gl.Project.PathWithNamespace,
				Description: desc,
				Footer:      glFooterForProject(gl.Project),
			},
		},
	}, nil
}

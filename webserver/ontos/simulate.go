//  Copyright (C) 2026 NodeByte LTD

package ontos

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/OctoHubOSS/Octoflow/webserver/logos/eventmodifiers"
	"github.com/OctoHubOSS/Octoflow/webserver/logos/events"
	"github.com/OctoHubOSS/Octoflow/webserver/state"
	"github.com/bwmarrin/discordgo"
	"github.com/go-chi/chi/v5"
)

var simulatableEvents = map[string]bool{
	"push":          true,
	"pull_request":  true,
	"issues":        true,
	"issue_comment": true,
	"release":       true,
	"star":          true,
	"fork":          true,
	"ping":          true,
}

type simulateRequest struct {
	GuildID string `json:"guild_id"`
	RepoID  string `json:"repo_id"`
	Event   string `json:"event"`
}

type simulateResponse struct {
	ACLFail  string                    `json:"acl_fail,omitempty"`
	Channels []string                  `json:"channels,omitempty"`
	Embeds   []*discordgo.MessageEmbed `json:"embeds,omitempty"`
}

func ApiDashboardSimulateEvent(w http.ResponseWriter, r *http.Request) {
	webhookId := chi.URLParam(r, "webhookId")

	var req simulateRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if req.GuildID == "" || req.RepoID == "" || req.Event == "" {
		writeError(w, http.StatusBadRequest, "guild_id, repo_id, and event are required")
		return
	}
	if !simulatableEvents[req.Event] {
		writeError(w, http.StatusBadRequest, "That event type can't be simulated. Supported: push, pull_request, issues, issue_comment, release, star, fork, ping")
		return
	}

	var repoName string
	err := state.Pool.QueryRow(
		state.Context,
		"SELECT repo_name FROM "+state.TableRepos+" WHERE id = $1 AND webhook_id = $2 AND guild_id = $3",
		req.RepoID, webhookId, req.GuildID,
	).Scan(&repoName)
	if err != nil {
		writeError(w, http.StatusNotFound, "That repo doesn't exist on this webhook")
		return
	}

	modres, err := eventmodifiers.CheckEventAllowed(webhookId, req.RepoID, req.Event)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error checking event modifiers: "+err.Error())
		return
	}

	resp := simulateResponse{}

	if modres.ACLFail != "" {
		resp.ACLFail = modres.ACLFail
		writeJSON(w, http.StatusOK, resp)
		return
	}

	if modres.ChannelOverride != "" {
		resp.Channels = []string{modres.ChannelOverride}
	} else {
		rows, err := state.Pool.Query(state.Context, "SELECT channel_id FROM "+state.TableRepos+" WHERE repo_name = $1 AND webhook_id = $2", strings.ToLower(repoName), webhookId)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Error resolving channels: "+err.Error())
			return
		}
		defer rows.Close()

		for rows.Next() {
			var channelId string
			if err := rows.Scan(&channelId); err != nil {
				continue
			}
			resp.Channels = append(resp.Channels, channelId)
		}
	}

	payload := buildSamplePayload(req.Event, repoName)

	evtFn, ok := events.SupportedEvents[req.Event]
	if !ok {
		writeError(w, http.StatusInternalServerError, "Internal error: event has no renderer")
		return
	}

	messageSend, err := evtFn(payload)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error rendering sample event: "+err.Error())
		return
	}

	resp.Embeds = messageSend.Embeds

	writeJSON(w, http.StatusOK, resp)
}

type sampleRepository struct {
	ID          int        `json:"id"`
	Name        string     `json:"name"`
	FullName    string     `json:"full_name"`
	Description string     `json:"description"`
	URL         string     `json:"url"`
	Owner       sampleUser `json:"owner"`
	HTMLURL     string     `json:"html_url"`
	CommitsURL  string     `json:"commits_url"`
	Private     bool       `json:"private"`
}

type sampleUser struct {
	Login            string `json:"login"`
	ID               int    `json:"id"`
	AvatarURL        string `json:"avatar_url"`
	URL              string `json:"url"`
	HTMLURL          string `json:"html_url"`
	OrganizationsURL string `json:"organizations_url"`
}

func sampleRepo(repoName string) sampleRepository {
	parts := strings.SplitN(repoName, "/", 2)
	owner := parts[0]
	name := repoName
	if len(parts) == 2 {
		name = parts[1]
	}

	return sampleRepository{
		ID:          1,
		Name:        name,
		FullName:    repoName,
		Description: "Sample repository for /testevent",
		URL:         "https://api.github.com/repos/" + repoName,
		HTMLURL:     "https://github.com/" + repoName,
		CommitsURL:  "https://api.github.com/repos/" + repoName + "/commits{/sha}",
		Private:     false,
		Owner: sampleUser{
			Login:     owner,
			ID:        1,
			AvatarURL: "https://avatars.githubusercontent.com/u/1?v=4",
			URL:       "https://api.github.com/users/" + owner,
			HTMLURL:   "https://github.com/" + owner,
		},
	}
}

func sampleSender() sampleUser {
	return sampleUser{
		Login:     "octocat",
		ID:        2,
		AvatarURL: "https://avatars.githubusercontent.com/u/2?v=4",
		URL:       "https://api.github.com/users/octocat",
		HTMLURL:   "https://github.com/octocat",
	}
}

func buildSamplePayload(event, repoName string) []byte {
	repo := sampleRepo(repoName)
	sender := sampleSender()

	var v any

	switch event {
	case "push":
		v = struct {
			Commits []struct {
				ID        string `json:"id"`
				Message   string `json:"message"`
				Timestamp string `json:"timestamp"`
				URL       string `json:"url"`
				Author    struct {
					Name     string `json:"name"`
					Email    string `json:"email"`
					Username string `json:"username"`
				} `json:"author"`
			} `json:"commits"`
			Repo   sampleRepository `json:"repository"`
			Sender sampleUser       `json:"sender"`
			Pusher struct {
				Name        string `json:"name"`
				Description string `json:"description"`
			} `json:"pusher,omitempty"`
			Ref     string `json:"ref"`
			BaseRef string `json:"base_ref"`
		}{
			Repo:    repo,
			Sender:  sender,
			Ref:     "refs/heads/main",
			BaseRef: "",
			Commits: []struct {
				ID        string `json:"id"`
				Message   string `json:"message"`
				Timestamp string `json:"timestamp"`
				URL       string `json:"url"`
				Author    struct {
					Name     string `json:"name"`
					Email    string `json:"email"`
					Username string `json:"username"`
				} `json:"author"`
			}{
				{
					ID:        "0123456789abcdef0123456789abcdef01234567",
					Message:   "Test commit from /testevent",
					Timestamp: time.Now().UTC().Format(time.RFC3339),
					URL:       "https://github.com/" + repoName + "/commit/0123456789abcdef0123456789abcdef01234567",
					Author: struct {
						Name     string `json:"name"`
						Email    string `json:"email"`
						Username string `json:"username"`
					}{Name: "octocat", Email: "octocat@github.com", Username: "octocat"},
				},
			},
			Pusher: struct {
				Name        string `json:"name"`
				Description string `json:"description"`
			}{Name: "octocat"},
		}

	case "pull_request":
		v = struct {
			Action      string           `json:"action"`
			Repo        sampleRepository `json:"repository"`
			Sender      sampleUser       `json:"sender"`
			PullRequest struct {
				ID      int            `json:"id"`
				Number  int            `json:"number"`
				State   string         `json:"state"`
				Locked  bool           `json:"locked"`
				Title   string         `json:"title"`
				Body    string         `json:"body"`
				HTMLURL string         `json:"html_url"`
				URL     string         `json:"url"`
				User    sampleUser     `json:"user"`
				Base    samplePRCommit `json:"base"`
				Head    samplePRCommit `json:"head"`
			} `json:"pull_request"`
		}{
			Action: "opened",
			Repo:   repo,
			Sender: sender,
			PullRequest: struct {
				ID      int            `json:"id"`
				Number  int            `json:"number"`
				State   string         `json:"state"`
				Locked  bool           `json:"locked"`
				Title   string         `json:"title"`
				Body    string         `json:"body"`
				HTMLURL string         `json:"html_url"`
				URL     string         `json:"url"`
				User    sampleUser     `json:"user"`
				Base    samplePRCommit `json:"base"`
				Head    samplePRCommit `json:"head"`
			}{
				ID: 1, Number: 1, State: "open", Title: "Test PR from /testevent",
				Body:    "This is a simulated pull_request event.",
				HTMLURL: "https://github.com/" + repoName + "/pull/1",
				User:    sender,
				Base:    samplePRCommit{Ref: "main", Label: repoName + ":main"},
				Head:    samplePRCommit{Ref: "feature", Label: repoName + ":feature"},
			},
		}

	case "issues":
		v = struct {
			Action string           `json:"action"`
			Repo   sampleRepository `json:"repository"`
			Sender sampleUser       `json:"sender"`
			Issue  sampleIssue      `json:"issue"`
		}{
			Action: "opened",
			Repo:   repo,
			Sender: sender,
			Issue:  sampleIssueValue(repoName, sender),
		}

	case "issue_comment":
		v = struct {
			Action  string           `json:"action"`
			Repo    sampleRepository `json:"repository"`
			Sender  sampleUser       `json:"sender"`
			Issue   sampleIssue      `json:"issue"`
			Comment struct {
				Body    string     `json:"body"`
				HTMLURL string     `json:"html_url"`
				User    sampleUser `json:"user"`
			} `json:"comment"`
		}{
			Action: "created",
			Repo:   repo,
			Sender: sender,
			Issue:  sampleIssueValue(repoName, sender),
			Comment: struct {
				Body    string     `json:"body"`
				HTMLURL string     `json:"html_url"`
				User    sampleUser `json:"user"`
			}{Body: "This is a simulated comment.", HTMLURL: "https://github.com/" + repoName + "/issues/1#issuecomment-1", User: sender},
		}

	case "release":
		v = struct {
			Action  string           `json:"action"`
			Repo    sampleRepository `json:"repository"`
			Sender  sampleUser       `json:"sender"`
			Release struct {
				HTMLUrl string `json:"html_url"`
				Body    string `json:"body"`
				TagName string `json:"tag_name"`
			} `json:"release"`
		}{
			Action: "published",
			Repo:   repo,
			Sender: sender,
			Release: struct {
				HTMLUrl string `json:"html_url"`
				Body    string `json:"body"`
				TagName string `json:"tag_name"`
			}{HTMLUrl: "https://github.com/" + repoName + "/releases/tag/v0.0.0-test", Body: "Simulated release notes.", TagName: "v0.0.0-test"},
		}

	case "star":
		v = struct {
			Action string           `json:"action"`
			Repo   sampleRepository `json:"repository"`
			Sender sampleUser       `json:"sender"`
		}{Action: "created", Repo: repo, Sender: sender}

	case "fork":
		forkee := repo
		forkee.FullName = sender.Login + "/" + repo.Name
		forkee.HTMLURL = "https://github.com/" + forkee.FullName
		v = struct {
			Action string           `json:"action"`
			Repo   sampleRepository `json:"repository"`
			Forkee sampleRepository `json:"forkee"`
			Sender sampleUser       `json:"sender"`
		}{Repo: repo, Forkee: forkee, Sender: sender}

	case "ping":
		v = struct {
			Zen    string           `json:"zen"`
			Repo   sampleRepository `json:"repository"`
			Sender sampleUser       `json:"sender"`
		}{Zen: "This is a simulated /testevent ping, not a real GitHub delivery.", Repo: repo, Sender: sender}
	}

	b, _ := json.Marshal(v)
	return b
}

type samplePRCommit struct {
	Ref   string `json:"ref"`
	Label string `json:"label"`
}

type sampleIssue struct {
	ID      int        `json:"id"`
	Number  int        `json:"number"`
	State   string     `json:"state"`
	Title   string     `json:"title"`
	Body    string     `json:"body"`
	HTMLURL string     `json:"html_url"`
	URL     string     `json:"url"`
	User    sampleUser `json:"user"`
}

func sampleIssueValue(repoName string, sender sampleUser) sampleIssue {
	return sampleIssue{
		ID: 1, Number: 1, State: "open",
		Title:   "Test issue from /testevent",
		Body:    "This is a simulated issues event.",
		HTMLURL: "https://github.com/" + repoName + "/issues/1",
		User:    sender,
	}
}

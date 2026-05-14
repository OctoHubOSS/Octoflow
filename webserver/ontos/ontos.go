// Ontos (Xenoblade Chronicles 2), the core component that recieves requests passing it down to
// Pneuma/Logos
package ontos

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/git-logs/client/webserver/logos/eventmodifiers"
	"github.com/git-logs/client/webserver/logos/events"
	"github.com/git-logs/client/webserver/pneuma"
	"github.com/git-logs/client/webserver/state"

	"github.com/infinitybotlist/eureka/crypto"
	"go.uber.org/zap"
)

func formatBool(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func GetWebhookRoute(w http.ResponseWriter, r *http.Request) {
	// Find the webhook in the database
	id := r.URL.Query().Get("id")

	if id == "" {
		w.WriteHeader(400)
		w.Write([]byte("This request is missing the id parameter"))
		return
	}

	var comment string
	var broken bool
	err := state.Pool.QueryRow(state.Context, "SELECT comment, broken FROM "+state.TableWebhooks+" WHERE id = $1", id).Scan(&comment, &broken)

	if err != nil {
		w.WriteHeader(404)
		w.Write([]byte("This request has an invalid id parameter"))
		return
	}

	var respStr = strings.Builder{}

	respStr.WriteString("Broken: " + formatBool(broken) + "\n")
	respStr.WriteString("Comment: " + comment + "\n\n")

	// Get all event modifiers on this webhook
	modifiers, err := eventmodifiers.GetEventModifiers(id, "")

	if err != nil {
		respStr.WriteString("ERROR: " + err.Error() + " in fetching event modifiers for webhook\n")
	} else {
		respStr.WriteString("EventModifiers:\n\n")

		for _, modifier := range modifiers {
			data := map[string]string{
				"ID":     modifier.ID,
				"Events": strings.Join(modifier.Events, ","),
				"RepoID": modifier.RepoID,
				"Blacklisted": func() string {
					if modifier.Blacklisted {
						return "true"
					}

					return "false"
				}(),
				"Whitelisted": func() string {
					if modifier.Whitelisted {
						return "true"
					}

					return "false"
				}(),
				"RedirectChannel": modifier.RedirectChannel,
				"Priority":        strconv.Itoa(modifier.Priority),
			}

			for k, v := range data {
				respStr.WriteString(k + ": " + v + "\n")
			}

			respStr.WriteString("\n")
		}

		respStr.WriteString("\n\n")
	}

	repos, err := state.Pool.Query(state.Context, "SELECT id, repo_name, channel_id, created_at FROM "+state.TableRepos+" WHERE webhook_id = $1", id)

	if err == nil {
		respStr.WriteString("Repositories:\n\n")

		for repos.Next() {
			var repoID string
			var repoName string
			var channelID string
			var createdAt time.Time

			err = repos.Scan(&repoID, &repoName, &channelID, &createdAt)

			if err != nil {
				respStr.WriteString("Error: " + err.Error() + " in fetching a repo \n")
				continue
			}

			respStr.WriteString("Repo: " + repoName + "\n")
			respStr.WriteString("Repo ID: " + repoID + "\n")
			respStr.WriteString("Channel ID: " + channelID + "\n")
			respStr.WriteString("Created At: " + createdAt.Format(time.RFC3339) + "\n\n")
		}
	} else {
		respStr.WriteString("This webhook doesn't seem to have any added repositories yet!\n")
	}

	w.WriteHeader(200)
	w.Write([]byte(respStr.String()))
}

func HandleWebhookRoute(w http.ResponseWriter, r *http.Request) {
	logId := crypto.RandString(128)

	id := r.URL.Query().Get("id")

	if id == "" {
		w.WriteHeader(400)
		w.Write([]byte("This request is missing the id parameter"))
		return
	}

	var secret string
	var broken bool
	var provider string
	err := state.Pool.QueryRow(state.Context, "SELECT secret, broken, COALESCE(provider, 'github') FROM "+state.TableWebhooks+" WHERE id = $1", id).Scan(&secret, &broken, &provider)

	if err != nil {
		w.WriteHeader(404)
		w.Write([]byte("This request has an invalid id parameter"))
		return
	}

	if broken {
		w.WriteHeader(500)
		w.Write([]byte("This webhook is marked as broken!"))
		return
	}

	var guildId string
	err = state.Pool.QueryRow(state.Context, "SELECT guild_id FROM "+state.TableWebhooks+" WHERE id = $1", id).Scan(&guildId)

	if err != nil {
		w.WriteHeader(404)
		w.Write([]byte("This request has an invalid id parameter"))
		return
	}

	var bodyBytes []byte

	defer r.Body.Close()
	if r.Body != nil {
		bodyBytes, _ = io.ReadAll(r.Body)
	}

	// Route to the appropriate provider handler
	switch provider {
	case "gitlab":
		handleGitLabWebhook(w, r, bodyBytes, id, logId, guildId, secret)
	default:
		handleGitHubWebhook(w, r, bodyBytes, id, logId, guildId, secret)
	}
}

func handleGitHubWebhook(w http.ResponseWriter, r *http.Request, bodyBytes []byte, id, logId, guildId, secret string) {
	var signature = r.Header.Get("X-Hub-Signature-256")

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(bodyBytes))
	expected := hex.EncodeToString(mac.Sum(nil))

	if "sha256="+expected != signature {
		w.WriteHeader(401)
		w.Write([]byte("This request has a bad signature, recheck the secret and ensure it isnt the id...."))
		return
	}

	if pneuma.NormalizeGitHubEventHeader(r.Header.Get("X-GitHub-Event")) == "ping" {
		w.WriteHeader(200)
		w.Write([]byte("pong"))
		return
	}

	var rw events.RepoWrapper

	err := json.Unmarshal(bodyBytes, &rw)

	if err != nil {
		state.Logger.Error("JSON unmarshal error", zap.Error(err))
		w.WriteHeader(400)
		w.Write([]byte("This request is not a valid JSON:" + err.Error()))
		return
	}

	var header = pneuma.NormalizeGitHubEventHeader(r.Header.Get("X-GitHub-Event"))

	// Get repo_name from database
	var repoName string
	var repoID string
	err = state.Pool.QueryRow(state.Context, "SELECT id, repo_name FROM "+state.TableRepos+" WHERE repo_name = $1 AND webhook_id = $2", strings.ToLower(rw.Repo.FullName), id).Scan(&repoID, &repoName)

	if err != nil {
		state.Logger.Warn("This repository is not configured on Octoflow, ignoring", zap.Error(err), zap.String("repoName", rw.Repo.FullName), zap.String("webhookID", id))
		w.WriteHeader(http.StatusPartialContent)
		w.Write([]byte("This repository is not configured on Octoflow, ignoring"))
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(
		"View logs at: " + state.Config.APIUrl + "/audit?log_id=" + logId + "\n" +
			"Going to process webhook event now: " + header + "\n",
	))

	go pneuma.HandleEvents(
		bodyBytes,
		&rw,
		repoID,
		logId,
		header,
		id,
		guildId,
		"github",
	)
}

func handleGitLabWebhook(w http.ResponseWriter, r *http.Request, bodyBytes []byte, id, logId, guildId, secret string) {
	// GitLab uses X-Gitlab-Token for verification (simple token comparison)
	token := r.Header.Get("X-Gitlab-Token")

	if token != secret {
		w.WriteHeader(401)
		w.Write([]byte("This request has a bad token, recheck the secret token in your GitLab webhook settings"))
		return
	}

	gitlabEvent := r.Header.Get("X-Gitlab-Event")

	if gitlabEvent == "" {
		w.WriteHeader(400)
		w.Write([]byte("Missing X-Gitlab-Event header"))
		return
	}

	// Parse the GitLab payload to get project info
	var glPayload struct {
		Project struct {
			PathWithNamespace string `json:"path_with_namespace"`
			WebURL            string `json:"web_url"`
		} `json:"project"`
		ObjectKind string `json:"object_kind"`
	}

	if err := json.Unmarshal(bodyBytes, &glPayload); err != nil {
		state.Logger.Error("GitLab JSON unmarshal error", zap.Error(err))
		w.WriteHeader(400)
		w.Write([]byte("This request is not valid JSON: " + err.Error()))
		return
	}

	repoFullName := strings.ToLower(glPayload.Project.PathWithNamespace)
	eventName := mapGitLabEventName(gitlabEvent, glPayload.ObjectKind)

	// Get repo_name from database
	var repoName string
	var repoID string
	err := state.Pool.QueryRow(state.Context, "SELECT id, repo_name FROM "+state.TableRepos+" WHERE repo_name = $1 AND webhook_id = $2", repoFullName, id).Scan(&repoID, &repoName)

	if err != nil {
		state.Logger.Warn("This repository is not configured on Octoflow, ignoring", zap.Error(err), zap.String("repoName", repoFullName), zap.String("webhookID", id))
		w.WriteHeader(http.StatusPartialContent)
		w.Write([]byte("This repository is not configured on Octoflow, ignoring"))
		return
	}

	// Create a synthetic RepoWrapper for GitLab (display path keeps webhook casing; DB match uses repoFullName).
	var rw events.RepoWrapper
	rw.Repo.FullName = strings.TrimSpace(glPayload.Project.PathWithNamespace)
	rw.Repo.HTMLURL = glPayload.Project.WebURL
	rw.Action = glPayload.ObjectKind

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(
		"View logs at: " + state.Config.APIUrl + "/audit?log_id=" + logId + "\n" +
			"Going to process GitLab webhook event now: " + eventName + "\n",
	))

	go pneuma.HandleEvents(
		bodyBytes,
		&rw,
		repoID,
		logId,
		eventName,
		id,
		guildId,
		"gitlab",
	)
}

// mapGitLabEventName maps GitLab event header values to internal event names
func mapGitLabEventName(headerEvent, objectKind string) string {
	switch headerEvent {
	case "Push Hook":
		return "gl_push"
	case "Tag Push Hook":
		return "gl_tag_push"
	case "Issue Hook":
		return "gl_issue"
	case "Note Hook":
		return "gl_note"
	case "Merge Request Hook":
		return "gl_merge_request"
	case "Pipeline Hook":
		return "gl_pipeline"
	case "Job Hook":
		return "gl_job"
	case "Deployment Hook":
		return "gl_deployment"
	case "Release Hook":
		return "gl_release"
	case "Wiki Page Hook":
		return "gl_wiki"
	default:
		return "gl_" + strings.ReplaceAll(strings.ToLower(headerEvent), " ", "_")
	}
}

func IndexPage(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`This is the API for the OctoFlow service. It handles webhooks from GitHub and as well as GitLab and sends them to Discord.

You may also be looking for:

- API (possibly unstable): api/
  - Counts: counts/
    - <server_count>,<user_count>,<shard_count>

- Webhooks: kittycat?id=ID
  - Get Webhook Info: GET kittycat?id=ID
  - Handle Github Webhook: POST kittycat?id=ID
  - Handle GitLab Webhook: POST kittycat?id=ID
   
`))

	w.Write([]byte(`[is_embedded]: ` + strconv.FormatBool(state.IsEmbedded) + "\n"))
}

func AuditEvent(w http.ResponseWriter, r *http.Request) {
	logId := r.URL.Query().Get("log_id")

	if logId == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Missing log_id parameter"))
		return
	}

	var log []string

	err := state.Pool.QueryRow(state.Context, "SELECT entries FROM "+state.TableWebhookLogs+" WHERE log_id = $1", logId).Scan(&log)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error getting log: " + err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(strings.Join(log, "\n")))
}

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	err := state.Pool.Ping(state.Context)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("unhealthy: " + err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

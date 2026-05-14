package events

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

// Semantic embed colors (GitHub-inspired, easier on the eyes than pure RGB primaries).
var (
	colorGreen   = 0x238636
	colorYellow  = 0xD29922
	colorRed     = 0xDA3633
	colorDarkRed = 0x8B0000
)

var SupportedEvents = map[string]func(bytes []byte) (*discordgo.MessageSend, error){
	"branch_protection_rule":         branchProtectionRuleFn,
	"check_suite":                    checkSuiteFn,
	"create":                         createFn,
	"issues":                         issuesFn,
	"issue_comment":                  issueCommentFn,
	"pull_request":                   pullRequestFn,
	"pull_request_review_comment":    pullRequestReviewCommentFn,
	"push":                           pushFn,
	"star":                           starFn,
	"status":                         statusFn,
	"release":                        releaseFn,
	"commit_comment":                 commitCommentFn,
	"deployment":                     deploymentFn,
	"deployment_status":              deploymentStatusFn,
	"discussion":                     discussionFn,
	"discussion_comment":             discussionCommentFn,
	"workflow_run":                   workflowRunFn,
	"dependabot_alert":               dependabotAlertFn,
	"delete":                         deleteFn,
	"workflow_job":                   workflowJobFn,
	"check_run":                      checkRunFn,
	"public":                         publicFn,
	"watch":                          watchFn,
	"repository":                     repositoryFn,
	"repository_vulnerability_alert": RepositoryVulnerabilityAlert,
	"team":                           teamFn,
	"fork":                           forkFn,
	"page_build":                     pageBuildFn,
	"code_scanning_alert":            codeScanningAlertFn,
	"secret_scanning_alert":          secretScanningAlertFn,
}

type User struct {
	Login            string `json:"login"`
	ID               int    `json:"id"`
	AvatarURL        string `json:"avatar_url"`
	URL              string `json:"url"`
	HTMLURL          string `json:"html_url"`
	OrganizationsURL string `json:"organizations_url"`
}

// GitHubWebURL returns a browser URL for this user or org. Webhook payloads often omit html_url
// on nested objects (for example organization), so we fall back to https://github.com/<login>.
func (u User) GitHubWebURL() string {
	if s := strings.TrimSpace(u.HTMLURL); s != "" {
		return s
	}
	login := strings.TrimSpace(u.Login)
	if login == "" {
		return ""
	}
	return "https://github.com/" + login
}

func (u User) AuthorEmbed() *discordgo.MessageEmbedAuthor {
	return &discordgo.MessageEmbedAuthor{
		Name:    u.Login,
		URL:     u.GitHubWebURL(),
		IconURL: u.AvatarURL,
	}
}

func (u User) Link() string {
	login := strings.TrimSpace(u.Login)
	if login == "" {
		return "—"
	}
	url := u.GitHubWebURL()
	if url == "" {
		return login
	}
	return "[" + strings.ReplaceAll(login, " ", "%20") + "](" + url + ")"
}

type Repository struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	URL         string `json:"url"`
	Owner       User   `json:"owner"`
	HTMLURL     string `json:"html_url"`
	CommitsURL  string `json:"commits_url"`
	Private     bool   `json:"private"`
}

// Commit returns the commit URL for the given commit ID.
func (r Repository) Commit(id string) string {
	id = strings.TrimSpace(id)
	if len(id) < 7 {
		return "—"
	}
	base := strings.TrimSpace(r.HTMLURL)
	if base == "" {
		return "`" + id[:7] + "`"
	}
	return "[" + id[:7] + "](" + base + "/commit/" + id + ")"
}

func (r Repository) Visibility() string {
	if r.Private {
		return "Private"
	}
	return "Public"
}

// MarkdownLink renders [full_name](html_url) for embed field text.
func (r Repository) MarkdownLink() string {
	name := strings.TrimSpace(r.FullName)
	if name == "" {
		name = strings.TrimSpace(r.Name)
	}
	if name == "" {
		return "—"
	}
	u := strings.TrimSpace(r.HTMLURL)
	if u == "" {
		return name
	}
	return "[" + name + "](" + u + ")"
}

// OwnerThumbnail uses the repository owner avatar when present.
func (r Repository) OwnerThumbnail() *discordgo.MessageEmbedThumbnail {
	return r.Owner.EmbedThumbnail()
}

// EmbedThumbnail uses this user's avatar as a Discord embed thumbnail.
func (u User) EmbedThumbnail() *discordgo.MessageEmbedThumbnail {
	if strings.TrimSpace(u.AvatarURL) == "" {
		return nil
	}
	return &discordgo.MessageEmbedThumbnail{URL: u.AvatarURL}
}

// CheckConclusionEmbedColor maps GitHub Actions / Checks conclusions to embed colors.
func CheckConclusionEmbedColor(conclusion string) int {
	switch strings.ToLower(strings.TrimSpace(conclusion)) {
	case "success", "fixed":
		return colorGreen
	case "failure", "cancelled", "timed_out", "timedout", "action_required":
		return colorRed
	case "skipped", "neutral", "stale":
		return colorYellow
	case "no conclusion yet!":
		return colorYellow
	default:
		if strings.TrimSpace(conclusion) == "" {
			return colorYellow
		}
		return colorGreen
	}
}

type Issue struct {
	ID      int    `json:"id"`
	Number  int    `json:"number"`
	State   string `json:"state"`
	Title   string `json:"title"`
	Body    string `json:"body"`
	HTMLURL string `json:"html_url"`
	URL     string `json:"url"`
	User    User   `json:"user"`
}

type PullRequestCommit struct {
	Repo       Repository `json:"repo"`
	ID         int        `json:"id"`
	Number     int        `json:"number"`
	State      string     `json:"state"`
	Title      string     `json:"title"`
	Body       string     `json:"body"`
	HTMLURL    string     `json:"html_url"`
	URL        string     `json:"url"`
	Ref        string     `json:"ref"`
	Label      string     `json:"label"`
	User       User       `json:"user"`
	CommitsURL string     `json:"commits_url"`
}

type PullRequest struct {
	ID      int               `json:"id"`
	Number  int               `json:"number"`
	State   string            `json:"state"`
	Locked  bool              `json:"locked"`
	Title   string            `json:"title"`
	Body    string            `json:"body"`
	HTMLURL string            `json:"html_url"`
	URL     string            `json:"url"`
	User    User              `json:"user"`
	Base    PullRequestCommit `json:"base"`
	Head    PullRequestCommit `json:"head"`
}

// Auxillary but useful for large lists of data
type KeyValue struct {
	Key   string
	Value any
}

func (k KeyValue) String() string {
	return k.Key + " => " + fmt.Sprint(k.Value)
}

func (k KeyValue) StringMD() string {
	return "**" + k.Key + "**" + " => " + fmt.Sprint(k.Value)
}

// Core struct for defining a basic github event
type RepoWrapper struct {
	Repo   Repository `json:"repository"`
	Action string     `json:"action"`
}

package events

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

var (
	colorGreen   = 0x00ff1a
	colorYellow  = 0xffff00
	colorRed     = 0xff0000
	colorDarkRed = 0x8b0000
)

var SupportedEvents = map[string]func(bytes []byte) (*discordgo.MessageSend, error){
	"branch_protection_rule":      branchProtectionRuleFn,
	"check_suite":                 checkSuiteFn,
	"create":                      createFn,
	"issues":                      issuesFn,
	"issue_comment":               issueCommentFn,
	"pull_request":                pullRequestFn,
	"pull_request_review_comment": pullRequestReviewCommentFn,
	"push":                        pushFn,
	"star":                        starFn,
	"status":                      statusFn,
	"release":                     releaseFn,
	"commit_comment":              commitCommentFn,
	"deployment":                  deploymentFn,
	"deployment_status":           deploymentStatusFn,
	"discussion":                  discussionFn,
	"discussion_comment":          discussionCommentFn,
	"workflow_run":                workflowRunFn,
	"dependabot_alert":            dependabotAlertFn,
	"delete":                      deleteFn,
	"workflow_job":                workflowJobFn,
	"check_run":                   checkRunFn,
	"public":                      publicFn,
	"watch":                       watchFn,
	"repository":                  repositoryFn,
	"team":                        teamFn,
	"fork":                        forkFn,
	"page_build":                  pageBuildFn,
	"code_scanning_alert":         codeScanningAlertFn,
	"secret_scanning_alert":       secretScanningAlertFn,
}

type User struct {
	Login            string `json:"login"`
	ID               int    `json:"id"`
	AvatarURL        string `json:"avatar_url"`
	URL              string `json:"url"`
	HTMLURL          string `json:"html_url"`
	OrganizationsURL string `json:"organizations_url"`
}

func (u User) AuthorEmbed() *discordgo.MessageEmbedAuthor {
	return &discordgo.MessageEmbedAuthor{
		Name:    u.Login,
		IconURL: u.AvatarURL,
	}
}

func (u User) Link() string {
	return "[" + strings.ReplaceAll(u.Login, " ", "%20") + "](" + u.HTMLURL + ")"
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
	return "[" + id[:7] + "](" + r.HTMLURL + "/commit/" + id + ")"
}

func (r Repository) Visibility() string {
	if r.Private {
		return "Private"
	}
	return "Public"
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

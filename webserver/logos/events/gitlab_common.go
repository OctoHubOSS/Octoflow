package events

import (
	"strings"

	"github.com/bwmarrin/discordgo"
)

// GitLab color palette
var (
	glColorOrange = 0xFC6D26
	glColorPurple = 0x6B4FBB
)

// GitLabSupportedEvents maps GitLab internal event names to handler functions
var GitLabSupportedEvents = map[string]func(bytes []byte) (*discordgo.MessageSend, error){
	"gl_push":          glPushFn,
	"gl_tag_push":      glTagPushFn,
	"gl_issue":         glIssueFn,
	"gl_note":          glNoteFn,
	"gl_merge_request": glMergeRequestFn,
	"gl_pipeline":      glPipelineFn,
	"gl_release":       glReleaseFn,
	"gl_wiki":          glWikiFn,
	"gl_deployment":    glDeploymentFn,
	"gl_job":           glJobFn,
}

// GLUser represents a GitLab user in webhook payloads
type GLUser struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

func (u GLUser) AuthorEmbed() *discordgo.MessageEmbedAuthor {
	return &discordgo.MessageEmbedAuthor{
		Name:    u.Name + " (@" + u.Username + ")",
		IconURL: u.AvatarURL,
	}
}

func (u GLUser) Link(baseURL string) string {
	if baseURL == "" {
		baseURL = "https://gitlab.com"
	}
	return "[" + strings.ReplaceAll(u.Username, " ", "%20") + "](" + baseURL + "/" + u.Username + ")"
}

// GLProject represents a GitLab project in webhook payloads
type GLProject struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	WebURL            string `json:"web_url"`
	AvatarURL         string `json:"avatar_url"`
	GitSSHURL         string `json:"git_ssh_url"`
	GitHTTPURL        string `json:"git_http_url"`
	Namespace         string `json:"namespace"`
	PathWithNamespace string `json:"path_with_namespace"`
	DefaultBranch     string `json:"default_branch"`
	Homepage          string `json:"homepage"`
	URL               string `json:"url"`
	SSHURL            string `json:"ssh_url"`
	HTTPURL           string `json:"http_url"`
	Visibility        string `json:"visibility_level,omitempty"`
}

// GLRepository represents a GitLab repository section in webhook payloads
type GLRepository struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Homepage    string `json:"homepage"`
}

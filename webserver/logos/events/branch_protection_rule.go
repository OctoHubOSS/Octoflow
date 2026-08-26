package events

import (
	"strings"

	"github.com/bwmarrin/discordgo"
)

type bprRule struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	UpdatedAt string `json:"updated_at"`
	CreatedAt string `json:"created_at"`

	// Settings
	AdminEnforced                            bool     `json:"admin_enforced"`
	RequireCodeOwnerReview                   bool     `json:"require_code_owner_review"`
	AllowDeletionsEnforcementLevel           string   `json:"allow_deletions_enforcement_level"`
	AllowForcePushesEnforcementLevel         string   `json:"allow_force_pushes_enforcement_level"`
	AuthorizedActorNames                     []string `json:"authorized_actor_names"`
	AuthorizedActorsOnly                     bool     `json:"authorized_actors_only"`
	AuthorizedDismissalActorsOnly            bool     `json:"authorized_dismissal_actors_only"`
	CreateProtected                          bool     `json:"create_protected"`
	DismissStaleReviewsOnPush                bool     `json:"dismiss_stale_reviews_on_push"`
	IgnoreApprovalsFromContributors          bool     `json:"ignore_approvals_from_contributors"`
	LinearHistoryRequirementEnforcementLevel string   `json:"linear_history_requirement_enforcement_level"`
	MergeQueueEnforcementLevel               string   `json:"merge_queue_enforcement_level"`
	PullRequestReviewsEnforcementLevel       string   `json:"pull_request_reviews_enforcement_level"`
	RequiredAapprovingReviewCount            int      `json:"required_approving_review_count"`
	RequiredConversationResolutionLevel      string   `json:"required_conversation_resolution_level"`
	RequiredDeploymentsEnforcementLevel      string   `json:"required_deployments_enforcement_level"`
	RequiredStatusChecks                     []string `json:"required_status_checks"`
	SignatureRequirementEnforcementLevel     string   `json:"signature_requirement_enforcement_level"`
	StrictRequiredStatusChecksPolicy         bool     `json:"strict_required_status_checks_policy"`
}

func (r bprRule) settings() string {
	// TODO, make the keys a bit more user friendly
	settings := []KeyValue{
		{
			Key:   "Admin enforced",
			Value: r.AdminEnforced,
		},
		{
			Key:   "Require code owner review",
			Value: r.RequireCodeOwnerReview,
		},
		{
			Key:   "Allow deletions",
			Value: r.AllowDeletionsEnforcementLevel,
		},
		{
			Key:   "Allow force pushes",
			Value: r.AllowForcePushesEnforcementLevel,
		},
		{
			Key:   "Authorized actors",
			Value: strings.Join(r.AuthorizedActorNames, ", "),
		},
		{
			Key:   "Authorized actors only",
			Value: r.AuthorizedActorsOnly,
		},
		{
			Key:   "Authorized dismissal actors only",
			Value: r.AuthorizedDismissalActorsOnly,
		},
		{
			Key:   "Create protected",
			Value: r.CreateProtected,
		},
		{
			Key:   "Dismiss stale reviews on push",
			Value: r.DismissStaleReviewsOnPush,
		},
		{
			Key:   "Ignore approvals from contributors",
			Value: r.IgnoreApprovalsFromContributors,
		},
		{
			Key:   "Linear history requirement",
			Value: r.LinearHistoryRequirementEnforcementLevel,
		},
		{
			Key:   "Merge queue requirement",
			Value: r.MergeQueueEnforcementLevel,
		},
		{
			Key:   "Pull request reviews requirement",
			Value: r.PullRequestReviewsEnforcementLevel,
		},
		{
			Key:   "Required approving review count",
			Value: r.RequiredAapprovingReviewCount,
		},
		{
			Key:   "Required conversation resolution",
			Value: r.RequiredConversationResolutionLevel,
		},
		{
			Key:   "Required deployments",
			Value: r.RequiredDeploymentsEnforcementLevel,
		},
		{
			Key:   "Required status checks",
			Value: strings.Join(r.RequiredStatusChecks, ", "),
		},
		{
			Key:   "Signature requirement",
			Value: r.SignatureRequirementEnforcementLevel,
		},
		{
			Key:   "Strict status checks",
			Value: r.StrictRequiredStatusChecksPolicy,
		},
	}

	settingsStr := ""

	for _, setting := range settings {
		if len(settingsStr) > 2500 {
			settingsStr += "\n..."
			break
		}

		settingsStr += setting.StringMD() + "\n"
	}

	return settingsStr
}

type BranchProtectionRuleEvent struct {
	Action  string         `json:"action"`
	Repo    Repository     `json:"repository"`
	Sender  User           `json:"sender"`
	Rule    bprRule        `json:"rule"`
	Changes map[string]any `json:"changes"`
}

func branchProtectionRuleFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh BranchProtectionRuleEvent

	// Unmarshal the JSON into our struct
	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	var color int
	var title string
	if gh.Action == "created" {
		color = colorGreen
		title = "New branch protection rule: " + gh.Repo.FullName
	} else if gh.Action == "edited" {
		color = colorYellow
		title = "Branch protection rule edited: " + gh.Repo.FullName
	} else {
		color = colorRed
		title = "Branch protection rule deleted: " + gh.Repo.FullName
	}

	desc := "**Settings:**\n\n" + gh.Rule.settings()

	changes := []string{}

	for k := range gh.Changes {
		changes = append(changes, k)
	}

	if len(changes) > 0 {
		desc += "\n\n**Changes:**\n\n" + strings.Join(changes, ", ")
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:       color,
				Timestamp:   nowTimestamp(),
				URL:         gh.Repo.HTMLURL,
				Title:       title,
				Author:      gh.Sender.AuthorEmbed(),
				Description: desc,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:  "User",
						Value: gh.Sender.Link(),
					},
				},
			},
		},
	}, nil
}

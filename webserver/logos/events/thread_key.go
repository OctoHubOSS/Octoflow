package events

// ThreadKey identifies which trackable issue/PR an event belongs to, for
// thread-per-PR/issue mode. Only a handful of event types carry an
// issue/PR number at all.
type ThreadKey struct {
	Number int
	Kind   string // "issue" or "pull_request"
	Title  string
	Action string
}

// threadKeySource is a deliberately minimal shape - just enough to extract
// the thread key without depending on each event type's full payload struct
// (IssuesEvent, PullRequestEvent, etc.), which live next to their *Fn
// renderers and aren't meant to be reused here.
type threadKeySource struct {
	Action string `json:"action"`
	Issue  *struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		// Non-null only when this "issue" is actually a pull request - GitHub
		// represents PRs as issues under the hood, and issue_comment fires
		// for both. Without this check a PR comment would be misfiled under
		// the "issue" thread kind instead of "pull_request".
		PullRequest *struct{} `json:"pull_request"`
	} `json:"issue"`
	PullRequest *struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
	} `json:"pull_request"`
}

// eventsWithThreadKey is the set of event types that can carry a thread key.
var eventsWithThreadKey = map[string]bool{
	"issues":                      true,
	"issue_comment":               true,
	"pull_request":                true,
	"pull_request_review":        true,
	"pull_request_review_comment": true,
}

// ExtractThreadKey pulls the issue/PR number, kind, title, and action out of
// a raw event body, for thread-per-PR/issue mode. Returns ok=false for event
// types that don't carry a thread key at all.
func ExtractThreadKey(header string, body []byte) (ThreadKey, bool) {
	if !eventsWithThreadKey[header] {
		return ThreadKey{}, false
	}

	var src threadKeySource
	if err := json.Unmarshal(body, &src); err != nil {
		return ThreadKey{}, false
	}

	if src.PullRequest != nil {
		return ThreadKey{
			Number: src.PullRequest.Number,
			Kind:   "pull_request",
			Title:  src.PullRequest.Title,
			Action: src.Action,
		}, true
	}

	if src.Issue != nil {
		kind := "issue"
		if src.Issue.PullRequest != nil {
			kind = "pull_request"
		}
		return ThreadKey{
			Number: src.Issue.Number,
			Kind:   kind,
			Title:  src.Issue.Title,
			Action: src.Action,
		}, true
	}

	return ThreadKey{}, false
}

// IsThreadOpeningEvent reports whether this event is the one that should
// create a new thread (as opposed to looking up an existing one).
func (k ThreadKey) IsThreadOpeningEvent(header string) bool {
	return k.Action == "opened" && (header == "issues" || header == "pull_request")
}

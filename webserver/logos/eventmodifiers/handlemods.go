package eventmodifiers

import (
	"strings"

	"github.com/OctoHubOSS/Octoflow/webserver/state"

	"github.com/jackc/pgx/v5/pgtype"
)

func isNull(s pgtype.Text) bool {
	return !s.Valid || s.String == ""
}

type EventCheck struct {
	ACLFail         string
	ChannelOverride string
	Overriden       bool
}

type EventModifier struct {
	ID              string
	RepoID          string
	Events          []string
	Blacklisted     bool
	Whitelisted     bool
	RedirectChannel string
	Priority        int
}

func GetEventModifiers(
	webhookId string,
	ghRepoId string,
) ([]*EventModifier, error) {
	rows, err := state.Pool.Query(state.Context, "SELECT id, repo_id, events, blacklisted, whitelisted, redirect_channel, priority FROM "+state.TableEventModifiers+" WHERE webhook_id = $1 ORDER BY priority DESC", webhookId)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var modifiers []*EventModifier

	for rows.Next() {
		var id string
		var repoId pgtype.Text
		var events []string
		var blacklisted bool
		var whitelisted bool
		var redirectChannel pgtype.Text
		var priority int

		err = rows.Scan(&id, &repoId, &events, &blacklisted, &whitelisted, &redirectChannel, &priority)

		if err != nil {
			return nil, err
		}

		if ghRepoId != "" && (!isNull(repoId) && repoId.String != ghRepoId) {
			continue
		}

		modifiers = append(modifiers, &EventModifier{
			ID:              id,
			RepoID:          repoId.String,
			Events:          events,
			Blacklisted:     blacklisted,
			Whitelisted:     whitelisted,
			RedirectChannel: redirectChannel.String,
			Priority:        priority,
		})
	}

	return modifiers, nil
}

func CheckEventAllowed(
	webhookId string,
	ghRepoId string,
	ghEvent string,
) (*EventCheck, error) {
	modifiers, err := GetEventModifiers(webhookId, ghRepoId)

	if err != nil {
		return nil, err
	}

	var resultantEventCheck *EventCheck = &EventCheck{}

	lowerGhEvent := strings.ToLower(ghEvent)

	for _, modifier := range modifiers {
		var matched bool
		for _, event := range modifier.Events {
			if isMatch(strings.ToLower(event), lowerGhEvent) {
				matched = true
				break
			}
		}

		if !matched {
			if modifier.Whitelisted {
				if resultantEventCheck.Overriden {
					return resultantEventCheck, nil
				}

				return &EventCheck{
					ACLFail: "event_modifier " + modifier.ID + ": whitelist-only event modifier but event not matched",
				}, nil
			}

			continue
		}

		if modifier.Blacklisted {
			return &EventCheck{
				ACLFail: "event_modifier " + modifier.ID + ": blacklisted event modifier and event matches modifier",
			}, nil
		}

		if modifier.RedirectChannel != "" {
			resultantEventCheck.ChannelOverride = modifier.RedirectChannel
			resultantEventCheck.Overriden = true
		}
	}

	return resultantEventCheck, nil
}

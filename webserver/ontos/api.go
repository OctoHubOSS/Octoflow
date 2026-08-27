//  Copyright (C) 2026 NodeByte LTD

package ontos

import (
	"net/http"
	"strings"

	"github.com/OctoHubOSS/Octoflow/webserver/logos/events"
)

var eventList []string

func init() {
	eventList = []string{}

	for event := range events.SupportedEvents {
		eventList = append(eventList, event)
	}
}

func ApiEventsListView(w http.ResponseWriter, r *http.Request) {
	events := []string{}

	for _, event := range eventList {
		events = append(events, "- "+event)
	}

	w.Write([]byte(strings.Join(events, "\n")))
}

func ApiEventsCommaSepView(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(strings.Join(eventList, ",")))
}

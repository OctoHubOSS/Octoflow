// Ontos (Xenoblade Chronicles 2), the core component that recieves requests passing it down to Pneuma/Logos
package ontos

import (
	"net/http"
	"strings"

	"github.com/OctoHubOSS/Octoflow/webserver/logos/events"
)

// Precomputed values
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

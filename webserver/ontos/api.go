// Ontos (Xenoblade Chronicles 2), the core component that recieves requests passing it down to Pneuma/Logos
package ontos

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/git-logs/client/webserver/logos/events"
	"github.com/git-logs/client/webserver/state"
)

// Precomputed values
var eventList []string
var glEventList []string

func init() {
	eventList = []string{}

	for event := range events.SupportedEvents {
		eventList = append(eventList, event)
	}

	glEventList = []string{}

	for event := range events.GitLabSupportedEvents {
		glEventList = append(glEventList, event)
	}
}

// This endpoint can only be used if the discordgo websocket is open
func ApiStats(w http.ResponseWriter, r *http.Request) {
	// TODO: Discord.State is always nil now, try adding a different way of calculating this
	if state.Discord.State == nil {
		w.Write([]byte("0,0,0"))
		return
	}

	// Get guild count
	guildCount := len(state.Discord.State.Guilds)
	var userCount int
	var shardCount = state.Discord.ShardCount

	for _, guild := range state.Discord.State.Guilds {
		userCount += guild.MemberCount
	}

	w.Write([]byte(fmt.Sprintf("%d,%d,%d", guildCount, userCount, shardCount)))
}

func ApiEventsListView(w http.ResponseWriter, r *http.Request) {
	ghEvents := []string{}

	for _, event := range eventList {
		ghEvents = append(ghEvents, "- "+event)
	}

	glEvents := []string{}

	for _, event := range glEventList {
		glEvents = append(glEvents, "- "+event)
	}

	w.Write([]byte("GitHub Events:\n" + strings.Join(ghEvents, "\n") + "\n\nGitLab Events:\n" + strings.Join(glEvents, "\n")))
}

func ApiEventsCommaSepView(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("github:" + strings.Join(eventList, ",") + "\ngitlab:" + strings.Join(glEventList, ",")))
}


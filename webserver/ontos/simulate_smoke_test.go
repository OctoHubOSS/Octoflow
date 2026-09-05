package ontos

import (
	"testing"

	"github.com/OctoHubOSS/Octoflow/webserver/logos/events"
)

// Verifies every buildSamplePayload case actually unmarshals correctly
// through the real events.SupportedEvents renderer - a JSON shape mismatch
// here would otherwise fail silently (an empty/wrong embed) rather than a
// compile error, since it's all just []byte at the boundary. No DB or
// network needed, so this is cheap to run on every change to either side.
func TestBuildSamplePayloadRenders(t *testing.T) {
	for eventType := range simulatableEvents {
		payload := buildSamplePayload(eventType, "octocat/hello-world")

		fn, ok := events.SupportedEvents[eventType]
		if !ok {
			t.Fatalf("%s: no renderer registered", eventType)
		}

		msg, err := fn(payload)
		if err != nil {
			t.Fatalf("%s: render error: %v", eventType, err)
		}

		if len(msg.Embeds) == 0 {
			t.Fatalf("%s: no embeds produced", eventType)
		}

		if msg.Embeds[0].Title == "" {
			t.Fatalf("%s: embed has empty title, payload shape probably wrong", eventType)
		}

		t.Logf("%s -> %q", eventType, msg.Embeds[0].Title)
	}
}

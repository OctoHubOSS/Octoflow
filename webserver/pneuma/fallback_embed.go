package pneuma

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// toJSONObject normalizes JSON-decoded objects to map[string]any. Some decoders or edge cases
// yield map types that do not match a plain type switch on map[string]any, so we also use reflection.
func toJSONObject(v any) (map[string]any, bool) {
	if v == nil {
		return nil, false
	}
	switch t := v.(type) {
	case map[string]any:
		return t, true
	default:
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Map && rv.Type().Key().Kind() == reflect.String {
			out := make(map[string]any, rv.Len())
			for _, k := range rv.MapKeys() {
				out[k.String()] = rv.MapIndex(k).Interface()
			}
			return out, true
		}
		return nil, false
	}
}

func markdownLink(label, url string) string {
	label = strings.TrimSpace(label)
	url = strings.TrimSpace(url)
	if label == "" {
		return "—"
	}
	if url == "" {
		return label
	}
	return "[" + strings.ReplaceAll(label, " ", "%20") + "](" + url + ")"
}

func appendNestedDiscordFields(out *[]*discordgo.MessageEmbedField, key string, v any) {
	m, ok := toJSONObject(v)
	if !ok {
		return
	}
	k := strings.ToLower(strings.TrimSpace(key))
	switch k {
	case "sender":
		if login, ok := m["login"].(string); ok && login != "" {
			*out = append(*out, &discordgo.MessageEmbedField{Name: "Sender", Value: login, Inline: true})
		}
	case "user":
		if login, ok := m["login"].(string); ok && login != "" {
			*out = append(*out, &discordgo.MessageEmbedField{Name: "User", Value: login, Inline: true})
		}
	case "owner":
		if login, ok := m["login"].(string); ok && login != "" {
			*out = append(*out, &discordgo.MessageEmbedField{Name: "Owner", Value: login, Inline: true})
		}
	case "pusher":
		if name, ok := m["name"].(string); ok && name != "" {
			*out = append(*out, &discordgo.MessageEmbedField{Name: "Pusher", Value: name, Inline: true})
		}
	case "repository", "project":
		if fn, ok := m["full_name"].(string); ok && fn != "" {
			html, _ := m["html_url"].(string)
			*out = append(*out, &discordgo.MessageEmbedField{Name: "Repository", Value: markdownLink(fn, html), Inline: true})
		} else if pn, ok := m["path_with_namespace"].(string); ok && pn != "" {
			web, _ := m["web_url"].(string)
			*out = append(*out, &discordgo.MessageEmbedField{Name: "Repository", Value: markdownLink(pn, web), Inline: true})
		} else if nm, ok := m["name"].(string); ok && nm != "" {
			*out = append(*out, &discordgo.MessageEmbedField{Name: "Repository", Value: nm, Inline: true})
		}
	case "organization", "org":
		login, _ := m["login"].(string)
		login = strings.TrimSpace(login)
		if login == "" {
			return
		}
		htmlURL, _ := m["html_url"].(string)
		url := strings.TrimSpace(htmlURL)
		if url == "" {
			url = "https://github.com/" + login
		}
		*out = append(*out, &discordgo.MessageEmbedField{
			Name:   "Organization",
			Value:  "[" + strings.ReplaceAll(login, " ", "%20") + "](" + url + ")",
			Inline: true,
		})
	case "label":
		name, _ := m["name"].(string)
		clr, _ := m["color"].(string)
		val := name
		switch {
		case name != "" && clr != "":
			val = name + " (`#" + clr + "`)"
		case clr != "":
			val = "`#" + clr + "`"
		}
		if val != "" {
			*out = append(*out, &discordgo.MessageEmbedField{Name: "Label", Value: val, Inline: true})
		}
	case "alert":
		var lines []string
		if s, ok := m["affected_package_name"].(string); ok && s != "" {
			lines = append(lines, "**Package:** "+s)
		}
		if s, ok := m["affected_range"].(string); ok && s != "" {
			lines = append(lines, "**Range:** "+s)
		}
		if s, ok := m["ghsa_id"].(string); ok && s != "" {
			lines = append(lines, "**GHSA:** "+s)
		}
		if s, ok := m["external_identifier"].(string); ok && s != "" {
			lines = append(lines, "**ID:** "+s)
		}
		if len(lines) > 0 {
			val := strings.Join(lines, "\n")
			if len(val) > 900 {
				val = val[:900] + "…"
			}
			*out = append(*out, &discordgo.MessageEmbedField{Name: "Alert", Value: val, Inline: false})
		}
	}
}

func scalarDiscordFieldValue(v any) (string, bool) {
	switch t := v.(type) {
	case string:
		return t, true
	case bool:
		return strconv.FormatBool(t), true
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10), true
		}
		return strconv.FormatFloat(t, 'f', -1, 64), true
	case json.Number:
		return t.String(), true
	case nil:
		return "", false
	default:
		return "", false
	}
}

// buildFallbackWebhookEmbedFields turns arbitrary webhook JSON into short embed fields without dumping
// raw Go map representations (which Discord users saw as unreadable "map[...]" text).
func buildFallbackWebhookEmbedFields(fields map[string]any) []*discordgo.MessageEmbedField {
	var out []*discordgo.MessageEmbedField

	for k, v := range fields {
		if _, ok := toJSONObject(v); ok {
			appendNestedDiscordFields(&out, k, v)
		}
	}

	seen := make(map[string]struct{})
	for _, f := range out {
		seen[strings.ToLower(f.Name)] = struct{}{}
	}

	for k, v := range fields {
		if _, ok := toJSONObject(v); ok {
			continue
		}
		s, ok := scalarDiscordFieldValue(v)
		if !ok || strings.TrimSpace(s) == "" {
			continue
		}
		name := cases.Title(language.English).String(strings.ReplaceAll(k, "_", " "))
		lname := strings.ToLower(name)
		if _, dup := seen[lname]; dup {
			continue
		}
		seen[lname] = struct{}{}
		val := s
		if len(val) > 200 {
			val = val[:200] + "…"
		}
		out = append(out, &discordgo.MessageEmbedField{Name: name, Value: val, Inline: true})
	}

	if len(out) == 0 {
		out = append(out, &discordgo.MessageEmbedField{
			Name:   "Notice",
			Value:  "No safe scalar fields were extracted from this payload (nested objects are omitted here).",
			Inline: false,
		})
	}
	if len(out) > EMBED_FIELDS_MAX_COUNT {
		out = out[:EMBED_FIELDS_MAX_COUNT]
	}
	return out
}

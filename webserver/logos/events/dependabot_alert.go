package events

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type DependabotAlertEvent struct {
	Action string     `json:"action"`
	Repo   Repository `json:"repository"`
	Sender User       `json:"sender"`
	Alert  struct {
		HTMLURL    string `json:"html_url"`
		State      string `json:"state"`
		Dependency struct {
			Package struct {
				Name      string `json:"name"`
				Ecosystem string `json:"ecosystem"`
			} `json:"package"`
			ManifestPath string `json:"manifest_path"`
			Scope        string `json:"scope"`
		} `json:"dependency"`
		SecurityAdvisory struct {
			Severity        string `json:"severity"`
			GHSAID          string `json:"ghsa_id"`
			CVEID           string `json:"cve_id"`
			Summary         string `json:"summary"`
			Description     string `json:"description"`
			Vulnerabilities []struct {
				Severity               string `json:"severity"`
				VulnerableVersionRange string `json:"vulnerable_version_range"`
				FirstPatchedVersion    struct {
					Identifier string `json:"identifier"`
				} `json:"first_patched_version"`
			} `json:"vulnerabilities"`
		} `json:"security_advisory"`
		DismissedReason string `json:"dismissed_reason"`
		DismissedBy     User   `json:"dismissed_by"`
	} `json:"alert"`
}

func dependabotAlertFn(bytes []byte) (*discordgo.MessageSend, error) {
	var gh DependabotAlertEvent

	err := json.Unmarshal(bytes, &gh)
	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	color := colorGreen
	if gh.Action == "closed" {
		color = colorRed
	}

	pkg := gh.Alert.Dependency.Package.Name
	eco := gh.Alert.Dependency.Package.Ecosystem
	depLine := fmt.Sprintf("**`%s`** · _%s_", pkg, eco)
	if gh.Alert.Dependency.ManifestPath != "" {
		depLine += "\n**Manifest:** `" + gh.Alert.Dependency.ManifestPath + "`"
	}
	if gh.Alert.Dependency.Scope != "" {
		depLine += "\n**Scope:** `" + gh.Alert.Dependency.Scope + "`"
		if gh.Alert.Dependency.Scope == "runtime" {
			depLine += " _(runtime deps are often exploitable at execution time)_"
		}
	}

	sev := gh.Alert.SecurityAdvisory.Severity
	if sev != "" {
		if sev == "high" || sev == "critical" {
			color = colorDarkRed
		} else if sev == "medium" {
			color = colorYellow
		}
	}

	advisoryBits := []string{}
	if gh.Alert.SecurityAdvisory.GHSAID != "" {
		advisoryBits = append(advisoryBits, "**GHSA:** `"+gh.Alert.SecurityAdvisory.GHSAID+"`")
	}
	if cve := strings.TrimSpace(gh.Alert.SecurityAdvisory.CVEID); cve != "" {
		if strings.HasPrefix(strings.ToUpper(cve), "CVE-") {
			advisoryBits = append(advisoryBits, "**CVE:** `"+cve+"`")
		} else {
			advisoryBits = append(advisoryBits, "**CVE:** `CVE-"+cve+"`")
		}
	}
	if sev != "" {
		advisoryBits = append(advisoryBits, "**Severity:** `"+sev+"`")
	}
	advisoryBlock := strings.Join(advisoryBits, "\n")
	if advisoryBlock == "" {
		advisoryBlock = "—"
	}

	summary := strings.TrimSpace(gh.Alert.SecurityAdvisory.Summary)
	if summary == "" {
		summary = "_No short summary from GitHub._"
	} else if len(summary) > 500 {
		summary = summary[:497] + "…"
	}

	body := strings.TrimSpace(gh.Alert.SecurityAdvisory.Description)
	if len(body) > 900 {
		body = body[:897] + "…"
	}
	if body != "" {
		summary += "\n\n" + body
	}

	var vulnLines []string
	for _, vuln := range gh.Alert.SecurityAdvisory.Vulnerabilities {
		line := ""
		if vuln.Severity != "" {
			line += "**" + vuln.Severity + "**"
		}
		if vuln.VulnerableVersionRange != "" {
			if line != "" {
				line += " · "
			}
			line += "`" + vuln.VulnerableVersionRange + "`"
		}
		if vuln.FirstPatchedVersion.Identifier != "" {
			line += "\n_Patched:_ `" + vuln.FirstPatchedVersion.Identifier + "`"
		}
		if strings.TrimSpace(line) != "" {
			vulnLines = append(vulnLines, line)
		}
	}
	vulns := strings.Join(vulnLines, "\n\n")
	if vulns == "" {
		vulns = "—"
	}

	dismissed := "—"
	if gh.Alert.DismissedReason != "" || gh.Alert.DismissedBy.Login != "" {
		dismissed = ""
		if gh.Alert.DismissedReason != "" {
			dismissed += "**Reason:** " + gh.Alert.DismissedReason
		}
		if gh.Alert.DismissedBy.Login != "" {
			if dismissed != "" {
				dismissed += "\n"
			}
			dismissed += "**By:** " + gh.Alert.DismissedBy.Link()
		}
	}

	alertURL := strings.TrimSpace(gh.Alert.HTMLURL)
	title := fmt.Sprintf("Dependabot · %s · %s", strings.TrimSpace(gh.Repo.FullName), gh.Alert.State)
	if strings.TrimSpace(gh.Repo.FullName) == "" {
		title = fmt.Sprintf("Dependabot · %s · %s", strings.TrimSpace(gh.Repo.Name), gh.Alert.State)
	}

	desc := fmt.Sprintf("**Package:** `%s` · **Ecosystem:** _%s_\n**State:** `%s` · **Action:** `%s`", pkg, eco, gh.Alert.State, gh.Action)
	if alertURL != "" {
		desc += "\n[**Open alert in GitHub**](" + alertURL + ")"
	}

	var thumb *discordgo.MessageEmbedThumbnail
	if gh.Repo.Owner.AvatarURL != "" {
		thumb = &discordgo.MessageEmbedThumbnail{URL: gh.Repo.Owner.AvatarURL}
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:       color,
				URL:         alertURL,
				Title:       title,
				Description: desc,
				Thumbnail:   thumb,
				Author:      gh.Sender.AuthorEmbed(),
				Fields: []*discordgo.MessageEmbedField{
					{Name: "Dependency", Value: truncateField(depLine, 1024), Inline: false},
					{Name: "Advisory", Value: truncateField(advisoryBlock, 1024), Inline: false},
					{Name: "Summary", Value: truncateField(summary, 1024), Inline: false},
					{Name: "Version ranges", Value: truncateField(vulns, 1024), Inline: false},
					{Name: "Dismissal", Value: truncateField(dismissed, 1024), Inline: false},
				},
			},
		},
	}, nil
}

package events

import (
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

	// Unmarshal the JSON into our struct
	err := json.Unmarshal(bytes, &gh)

	if err != nil {
		return &discordgo.MessageSend{}, err
	}

	var color int
	if gh.Action == "closed" {
		color = colorRed
	} else {
		color = colorGreen
	}

	var details = gh.Alert.Dependency.Package.Name + " (" + gh.Alert.Dependency.Package.Ecosystem + ")"

	if gh.Alert.Dependency.Scope != "" {
		details += "\n**Scope:** " + gh.Alert.Dependency.Scope

		if gh.Alert.Dependency.Scope == "runtime" {
			details += " (this could be a highly critical vulnerability; runtime dependencies may not be checked by Dependabot)"
		}
	}

	if gh.Alert.Dependency.ManifestPath != "" {
		details += "\n**Manifest Path:** " + gh.Alert.Dependency.ManifestPath
	}

	if gh.Alert.SecurityAdvisory.Severity != "" {
		details += "\n**Severity:** " + gh.Alert.SecurityAdvisory.Severity

		if gh.Alert.SecurityAdvisory.Severity == "high" || gh.Alert.SecurityAdvisory.Severity == "critical" {
			color = colorDarkRed
		}
	}

	if gh.Alert.SecurityAdvisory.GHSAID != "" {
		details += "\n**GHSA ID:** " + gh.Alert.SecurityAdvisory.GHSAID
	}

	if gh.Alert.SecurityAdvisory.CVEID != "" {
		details += "\n**CVE:** CVE " + gh.Alert.SecurityAdvisory.CVEID
	}

	if gh.Alert.State == "fixed" {
		details += "\n**Could be fixed by resolving:** " + gh.Alert.Dependency.Package.Name + " " + gh.Alert.SecurityAdvisory.GHSAID
	}

	details = truncateText(details, 1020)

	var summaryDet string

	if gh.Alert.SecurityAdvisory.Summary != "" {
		summaryDet += "\n**Summary:** " + gh.Alert.SecurityAdvisory.Summary
	}

	if gh.Alert.SecurityAdvisory.Description != "" {
		summaryDet += "\n\n" + truncateText(gh.Alert.SecurityAdvisory.Description, 996)
	}

	summaryDet = truncateText(summaryDet, 1020)

	var vulns string

	for _, vuln := range gh.Alert.SecurityAdvisory.Vulnerabilities {
		vulns += "\n**Severity:** " + vuln.Severity + "\n**Vulnerable Version Range:** " + vuln.VulnerableVersionRange + "\n**First Patched Version:** " + vuln.FirstPatchedVersion.Identifier + "\n"
	}

	vulns = truncateText(vulns, 1020)

	var dismissed string

	if gh.Alert.DismissedReason != "" {
		dismissed += "\n**Dismissed Reason:** " + gh.Alert.DismissedReason
	}

	dismissed = truncateText(dismissed, 1020)

	if gh.Alert.DismissedBy.Login != "" {
		dismissed += "\n**Dismissed By:** " + gh.Alert.DismissedBy.Link()
	}

	if dismissed == "" {
		dismissed = "Not dismissed"
	}

	return &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Color:     color,
				Timestamp: nowTimestamp(),
				URL:       gh.Alert.HTMLURL,
				Title:     "Dependabot Alert on " + gh.Repo.FullName + " " + gh.Alert.State,
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "URL",
						Value:  gh.Alert.HTMLURL,
						Inline: true,
					},
					{
						Name:   "State",
						Value:  gh.Alert.State,
						Inline: true,
					},
					{
						Name:   "Details",
						Value:  details,
						Inline: true,
					},
					{
						Name:   "Summary",
						Value:  summaryDet,
						Inline: true,
					},
					{
						Name:   "Vulnerabilities",
						Value:  vulns,
						Inline: true,
					},
					{
						Name:   "Dismissal Details",
						Value:  dismissed,
						Inline: true,
					},
				},
			},
		},
	}, nil
}

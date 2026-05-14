package events

import (
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

func providerDisplayName(provider string) string {
	if strings.TrimSpace(provider) == "gitlab" {
		return "GitLab"
	}
	return "GitHub"
}

// repoMarkdownSegment is a Discord markdown link for the repo when html_url is present.
func repoMarkdownSegment(rw *RepoWrapper) string {
	if rw == nil {
		return ""
	}
	s := strings.TrimSpace(rw.Repo.MarkdownLink())
	if strings.Contains(s, "](") {
		return s
	}
	return ""
}

// prependRepoMarkdownOnce adds a clickable repo line to the embed description (footers do not support links).
func prependRepoMarkdownOnce(e *discordgo.MessageEmbed, md, fullName string) {
	if e == nil || md == "" {
		return
	}
	desc := e.Description
	trimDesc := strings.TrimSpace(desc)
	if trimDesc == "" {
		e.Description = md
		return
	}
	if strings.HasPrefix(trimDesc, md) {
		return
	}
	if idx := strings.Index(md, "]("); idx > 0 {
		prefix := md[:idx+len("](")]
		if strings.Contains(desc, prefix) {
			return
		}
	}
	if fullName != "" && strings.Contains(desc, "["+fullName+"](") {
		return
	}
	e.Description = md + "\n\n" + desc
}

// footerWithoutLeadingRepo removes a duplicated repo path from footer text so it matches the markdown line in description.
func footerWithoutLeadingRepo(footerText, repo, provider string) string {
	t := strings.TrimSpace(footerText)
	repo = strings.TrimSpace(repo)
	pr := providerDisplayName(provider)
	if repo == "" {
		if t == "" {
			return pr
		}
		return t
	}
	parts := strings.Split(t, " · ")
	if len(parts) >= 2 {
		left := strings.TrimSpace(parts[0])
		if strings.EqualFold(left, repo) {
			rest := strings.TrimSpace(strings.Join(parts[1:], " · "))
			if rest == "" {
				return pr
			}
			return rest
		}
	}
	suff := " · " + repo
	if len(t) >= len(suff) && strings.EqualFold(t[len(t)-len(suff):], suff) {
		out := strings.TrimSpace(t[:len(t)-len(suff)])
		if out == "" {
			return pr
		}
		return out
	}
	if strings.EqualFold(t, repo) {
		return pr
	}
	if t == "" {
		return pr
	}
	return t
}

// StripPlainRepoFromTitle removes the repository path from a title so the clickable line in the
// description is not duplicated (Discord titles do not support inline markdown links).
func StripPlainRepoFromTitle(title, repo string) string {
	title = strings.TrimSpace(title)
	repo = strings.TrimSpace(repo)
	if title == "" || repo == "" {
		return title
	}
	if strings.EqualFold(title, repo) {
		return ""
	}
	parts := strings.Split(title, " · ")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if strings.EqualFold(p, repo) {
			continue
		}
		out = append(out, p)
	}
	res := strings.Join(out, " · ")
	suff := ": " + repo
	if len(res) >= len(suff) && strings.EqualFold(res[len(res)-len(suff):], suff) {
		res = strings.TrimSpace(res[:len(res)-len(suff)])
	}
	suff2 := " on " + repo
	if len(res) >= len(suff2) && strings.EqualFold(res[len(res)-len(suff2):], suff2) {
		res = strings.TrimSpace(res[:len(res)-len(suff2)])
	}
	suff3 := " in " + repo
	if len(res) >= len(suff3) && strings.EqualFold(res[len(res)-len(suff3):], suff3) {
		res = strings.TrimSpace(res[:len(res)-len(suff3)])
	}
	return strings.TrimSpace(res)
}

// EnrichEmbedsBeforeSend fills missing timestamps and normalizes footers so Discord embeds look
// consistent across GitHub and GitLab handlers without editing every event file.
func EnrichEmbedsBeforeSend(embeds []*discordgo.MessageEmbed, rw *RepoWrapper, provider string) {
	if len(embeds) == 0 || rw == nil {
		return
	}

	ts := time.Now().UTC().Format(time.RFC3339)
	repo := strings.TrimSpace(rw.Repo.FullName)
	md := repoMarkdownSegment(rw)
	pr := providerDisplayName(provider)

	for _, e := range embeds {
		if e == nil {
			continue
		}
		if e.Timestamp == "" {
			e.Timestamp = ts
		}

		if e.Thumbnail == nil && rw.Repo.Owner.AvatarURL != "" {
			e.Thumbnail = &discordgo.MessageEmbedThumbnail{URL: rw.Repo.Owner.AvatarURL}
		}

		if md != "" {
			prependRepoMarkdownOnce(e, md, repo)
			if e.Footer != nil {
				e.Footer.Text = footerWithoutLeadingRepo(e.Footer.Text, repo, provider)
				if strings.TrimSpace(e.Footer.Text) == "" {
					e.Footer.Text = pr
				}
			}
			if strings.TrimSpace(e.Title) != "" {
				if t := StripPlainRepoFromTitle(e.Title, repo); t != "" {
					e.Title = t
				} else {
					e.Title = pr
				}
			}
			linkVal := strings.TrimSpace(rw.Repo.MarkdownLink())
			for _, f := range e.Fields {
				if f == nil {
					continue
				}
				if strings.EqualFold(strings.TrimSpace(f.Value), repo) {
					f.Value = linkVal
				}
			}
		}

		if e.Footer == nil {
			ft := &discordgo.MessageEmbedFooter{}
			switch {
			case md != "":
				ft.Text = pr
				if rw.Repo.Owner.AvatarURL != "" {
					ft.IconURL = rw.Repo.Owner.AvatarURL
				}
			case repo != "":
				ft.Text = repo
				if rw.Repo.Owner.AvatarURL != "" {
					ft.IconURL = rw.Repo.Owner.AvatarURL
				}
			case provider == "gitlab":
				ft.Text = "GitLab"
			default:
				ft.Text = "GitHub"
			}
			e.Footer = ft
			continue
		}

		if md != "" {
			continue
		}

		if repo != "" && !strings.Contains(e.Footer.Text, repo) {
			if len(e.Footer.Text)+len(repo)+3 <= 2000 {
				e.Footer.Text = e.Footer.Text + " · " + repo
			}
		}
	}
}

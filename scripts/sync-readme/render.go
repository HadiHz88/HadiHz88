package main

import (
	"errors"
	"fmt"
	"html"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

var mdEscaper = strings.NewReplacer(
	`\`, `\\`, `|`, `\|`, `[`, `\[`, `]`, `\]`, `<`, `&lt;`, `>`, `&gt;`,
)

func md(s string) string { return mdEscaper.Replace(strings.TrimSpace(s)) }

// safeURL drops anything that is not an absolute http(s) link, so CMS input
// cannot inject javascript: links or break out of the Markdown link syntax.
func safeURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return ""
	}
	return strings.NewReplacer("(", "%28", ")", "%29", " ", "%20").Replace(u.String())
}

func link(text, href string) string {
	if u := safeURL(href); u != "" {
		return "[" + text + "](" + u + ")"
	}
	return text
}

func parseDate(s string) (time.Time, bool) {
	t, err := time.Parse("2006-01-02", s)
	return t, err == nil
}

func monthYear(s string) string {
	if t, ok := parseDate(s); ok {
		return t.Format("Jan 2006")
	}
	return ""
}

func inProgress(end string, now time.Time) bool {
	t, ok := parseDate(end)
	return !ok || t.After(now)
}

func dateRange(start, end string, now time.Time) string {
	from := monthYear(start)
	to := "Present"
	if !inProgress(end, now) {
		to = monthYear(end)
	}
	if from == "" {
		return to
	}
	return from + " – " + to
}

func title(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

var roleEmoji = map[string]string{
	"full-time":  "💼",
	"part-time":  "🧑‍🏫",
	"internship": "🌱",
	"freelance":  "🧩",
	"contract":   "📝",
}

// renderNow lists the featured roles that have not ended, newest first.
func renderNow(items []Experience, now time.Time) (string, error) {
	var cur []Experience
	for _, e := range items {
		if e.Featured && inProgress(e.EndDate, now) {
			cur = append(cur, e)
		}
	}
	if len(cur) == 0 {
		return "", errors.New("no featured experience in progress")
	}
	sort.SliceStable(cur, func(i, j int) bool { return cur[i].StartDate > cur[j].StartDate })

	var b strings.Builder
	for _, e := range cur {
		emoji := roleEmoji[e.EmploymentType]
		if emoji == "" {
			emoji = "💼"
		}
		fmt.Fprintf(&b, "- %s **%s**", emoji, md(e.Title))
		if e.Company != "" {
			fmt.Fprintf(&b, " @ %s", link("**"+md(e.Company)+"**", e.URL))
		}
		meta := []string{}
		if e.EmploymentType != "" {
			meta = append(meta, title(e.EmploymentType))
		}
		if since := monthYear(e.StartDate); since != "" {
			meta = append(meta, "since "+since)
		}
		if e.Location != "" {
			meta = append(meta, md(e.Location))
		}
		if len(meta) > 0 {
			fmt.Fprintf(&b, " · <sub>%s</sub>", strings.Join(meta, " · "))
		}
		if d := strings.Join(strings.Fields(e.Description), " "); d != "" {
			fmt.Fprintf(&b, "\n  <br><sub>%s</sub>", md(d))
		}
		b.WriteString("\n")
	}
	return b.String(), nil
}

func renderEducation(items []Education, now time.Time) (string, error) {
	var list []Education
	for _, e := range items {
		if e.Featured {
			list = append(list, e)
		}
	}
	if len(list) == 0 {
		return "", errors.New("no featured education")
	}
	sort.SliceStable(list, func(i, j int) bool { return list[i].StartDate > list[j].StartDate })

	var b strings.Builder
	for _, e := range list {
		fmt.Fprintf(&b, "- 🎓 **%s**", md(e.Title))
		if e.Institution != "" {
			fmt.Fprintf(&b, " · %s", link(md(e.Institution), e.URL))
		}
		meta := []string{dateRange(e.StartDate, e.EndDate, now)}
		if g := strings.TrimSpace(e.Grade); g != "" {
			meta = append(meta, md(g))
		}
		fmt.Fprintf(&b, " · <sub>%s</sub>\n", strings.Join(meta, " · "))
	}
	return b.String(), nil
}

var skillGroups = []struct{ category, heading string }{
	{"language", "Languages"},
	{"frontend", "Frontend"},
	{"backend", "Backend"},
	{"databases", "Databases & ORMs"},
	{"devops", "Cloud & DevOps"},
	{"tools", "Tools"},
}

func renderSkills(tags []Tag) (string, error) {
	if len(tags) == 0 {
		return "", errors.New("no skills")
	}
	sort.SliceStable(tags, func(i, j int) bool {
		a, b := tags[i], tags[j]
		if a.Featured != b.Featured {
			return a.Featured
		}
		if a.Level != b.Level {
			return a.Level > b.Level
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})
	by := map[string][]Tag{}
	for _, t := range tags {
		by[t.Category] = append(by[t.Category], t)
	}

	var b strings.Builder
	for _, g := range skillGroups {
		list := by[g.category]
		if len(list) == 0 {
			continue
		}
		badges := make([]string, len(list))
		for i, t := range list {
			badges[i] = badge(t)
		}
		fmt.Fprintf(&b, "### %s\n\n%s\n\n", g.heading, strings.Join(badges, " "))
	}
	// Concepts rather than tools: their CMS icons are stand-ins, so plain text reads better.
	if other := by["other"]; len(other) > 0 {
		names := make([]string, len(other))
		for i, t := range other {
			names[i] = md(t.Name)
		}
		fmt.Fprintf(&b, "<sub>**Also:** %s</sub>\n", strings.Join(names, " · "))
	}
	return b.String(), nil
}

var shieldsEscaper = strings.NewReplacer("-", "--", "_", "__")

func badge(t Tag) string {
	color, logo := "555555", ""
	if t.Icon != nil {
		if c := strings.TrimPrefix(t.Icon.Color, "#"); isHex(c) {
			color = strings.ToUpper(c)
		}
		if slug, ok := strings.CutPrefix(t.Icon.Name, "simple-icons:"); ok {
			logo = slug
		}
	}
	label := url.PathEscape(shieldsEscaper.Replace(t.Name))
	src := "https://img.shields.io/badge/-" + label + "-" + color + "?style=for-the-badge"
	if logo != "" {
		src += "&logo=" + url.QueryEscape(logo) + "&logoColor=" + logoColor(color)
	}
	return "![" + md(t.Name) + "](" + src + ")"
}

func isHex(s string) bool {
	if len(s) != 6 {
		return false
	}
	_, err := strconv.ParseUint(s, 16, 32)
	return err == nil
}

// logoColor picks black or white for contrast using perceived luminance.
func logoColor(hex string) string {
	v, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return "white"
	}
	r, g, b := float64(v>>16&0xff), float64(v>>8&0xff), float64(v&0xff)
	if 0.299*r+0.587*g+0.114*b > 160 {
		return "black"
	}
	return "white"
}

var statusRank = map[string]int{"active": 0, "planned": 1, "completed": 2, "archived": 3, "canceled": 4}

const maxProjects = 6

func selectProjects(items []Project) []Project {
	var list []Project
	for _, p := range items {
		if p.Featured {
			list = append(list, p)
		}
	}
	sort.SliceStable(list, func(i, j int) bool {
		a, b := list[i], list[j]
		if ra, rb := statusRank[a.Status], statusRank[b.Status]; ra != rb {
			return ra < rb
		}
		if a.StartDate != b.StartDate {
			return a.StartDate > b.StartDate // empty dates sort last
		}
		return a.Title < b.Title
	})
	if len(list) > maxProjects {
		list = list[:maxProjects]
	}
	return list
}

// renderProjects returns the README block plus the SVG card files it references.
func renderProjects(items []Project, icons map[string]iconSVG, assetDir, site string) (string, map[string][]byte, error) {
	list := selectProjects(items)
	if len(list) == 0 {
		return "", nil, errors.New("no featured projects")
	}
	files := map[string][]byte{}
	var b strings.Builder
	b.WriteString("<p align=\"center\">\n")
	for i, p := range list {
		slug := slugify(p.Title)
		dark := path.Join(assetDir, slug+"-dark.svg")
		light := path.Join(assetDir, slug+"-light.svg")
		files[dark] = card(p, darkTheme, icons)
		files[light] = card(p, lightTheme, icons)

		href := strings.TrimRight(site, "/") + "/projects/" + slug
		alt := html.EscapeString(strings.TrimSpace(p.Title + " — " + p.Summary))
		fmt.Fprintf(&b,
			"<a href=\"%s\"><picture><source media=\"(prefers-color-scheme: dark)\" srcset=\"%s\"><img alt=\"%s\" src=\"%s\" width=\"49%%\"></picture></a>",
			html.EscapeString(href), dark, alt, light)
		if i%2 == 1 || i == len(list)-1 {
			b.WriteString("\n")
		} else {
			b.WriteString(" ")
		}
	}
	b.WriteString("</p>\n")
	return b.String(), files, nil
}

// slugify mirrors the site's project slugs (apps/web core/content/collection.ts),
// minus Unicode normalisation, which the stdlib lacks.
func slugify(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.TrimRight(b.String(), "-")
}

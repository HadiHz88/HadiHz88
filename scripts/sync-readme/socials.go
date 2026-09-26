package main

import (
	"context"
	"fmt"
	"html"
	"net/url"
	"path"
	"regexp"
	"strings"
)

type Profile struct {
	Headline    string `json:"headline"`
	Position    string `json:"position"`
	Description string `json:"description"`
	City        string `json:"city"`
	Country     string `json:"country"`
	Email       string `json:"email"`
	GitHub      string `json:"github"`
	LinkedIn    string `json:"linkedin"`
	YouTube     string `json:"youtube"`
	Instagram   string `json:"instagram"`
}

type social struct {
	key, name, icon, color string
	hosts                  []string // accepted hosts, so a mis-pasted URL never ships
	href, handle           string
}

func fetchProfile(ctx context.Context, c *client) (Profile, error) {
	var res struct {
		Data Profile `json:"data"`
	}
	// An explicit field list keeps phone number and date of birth out of the response.
	err := c.get(ctx, "/api/profile", query(
		"fields[0]", "headline", "fields[1]", "position", "fields[2]", "description",
		"fields[3]", "city", "fields[4]", "country", "fields[5]", "email",
		"fields[6]", "github", "fields[7]", "linkedin", "fields[8]", "youtube",
		"fields[9]", "instagram",
	), &res)
	return res.Data, err
}

var trailingID = regexp.MustCompile(`-[a-z0-9]*\d[a-z0-9]*$`)

// socialLinks returns the valid links in display order plus a note per rejected one.
func socialLinks(p Profile) ([]social, []string) {
	all := []social{
		{key: "github", name: "GitHub", icon: "simple-icons:github", color: "#181717", hosts: []string{"github.com"}, href: p.GitHub},
		{key: "linkedin", name: "LinkedIn", icon: "simple-icons:linkedin", color: "#0A66C2", hosts: []string{"linkedin.com"}, href: p.LinkedIn},
		{key: "youtube", name: "YouTube", icon: "simple-icons:youtube", color: "#FF0000", hosts: []string{"youtube.com", "youtu.be"}, href: p.YouTube},
		{key: "instagram", name: "Instagram", icon: "simple-icons:instagram", color: "#E4405F", hosts: []string{"instagram.com"}, href: p.Instagram},
	}
	var out []social
	var notes []string
	for _, s := range all {
		if strings.TrimSpace(s.href) == "" {
			continue
		}
		u, err := url.Parse(safeURL(s.href))
		if err != nil || u.Host == "" || !hostMatches(u.Host, s.hosts) {
			notes = append(notes, fmt.Sprintf("%s link %q is not a %s URL", s.name, s.href, s.hosts[0]))
			continue
		}
		u.Scheme, u.RawQuery, u.Fragment = "https", "", ""
		s.href = u.String()
		s.handle = handleOf(s.key, u)
		out = append(out, s)
	}
	if e := strings.TrimSpace(p.Email); strings.Count(e, "@") == 1 && !strings.ContainsAny(e, " <>\"") {
		out = append(out, social{key: "email", name: "Email", icon: "simple-icons:gmail", color: "#EA4335", href: "mailto:" + e, handle: e})
	}
	return out, notes
}

func hostMatches(host string, hosts []string) bool {
	host = strings.TrimPrefix(strings.ToLower(host), "www.")
	host = strings.TrimPrefix(host, "m.")
	for _, h := range hosts {
		if host == h {
			return true
		}
	}
	return false
}

func handleOf(key string, u *url.URL) string {
	seg := strings.Trim(u.Path, "/")
	if i := strings.LastIndex(seg, "/"); i >= 0 {
		seg = seg[i+1:]
	}
	switch key {
	case "linkedin":
		words := strings.Split(trailingID.ReplaceAllString(seg, ""), "-")
		for i, w := range words {
			words[i] = title(w)
		}
		return strings.Join(words, " ")
	case "youtube", "instagram":
		return "@" + strings.TrimPrefix(seg, "@")
	}
	return seg
}

// renderSocials fills SOCIALS with every link and CONTACT (the footer) with the core three.
func renderSocials(links []social, icons map[string]iconSVG, assetDir string) (map[string]string, map[string][]byte, error) {
	if len(links) == 0 {
		return nil, nil, fmt.Errorf("no valid social links")
	}
	files := map[string][]byte{}
	tags := map[string]string{}
	for _, s := range links {
		dark := path.Join(assetDir, s.key+"-dark.svg")
		light := path.Join(assetDir, s.key+"-light.svg")
		files[dark] = socialButton(s, darkTheme, icons)
		files[light] = socialButton(s, lightTheme, icons)
		tags[s.key] = fmt.Sprintf(
			"<a href=\"%s\"><picture><source media=\"(prefers-color-scheme: dark)\" srcset=\"%s\"><img alt=\"%s\" src=\"%s\"></picture></a>",
			html.EscapeString(s.href), dark, html.EscapeString(s.name+" — "+s.handle), light)
	}
	row := func(keys []string) string {
		var parts []string
		for _, k := range keys {
			if t, ok := tags[k]; ok {
				parts = append(parts, t)
			}
		}
		// More than four wraps unevenly at README width, so split into two balanced rows.
		if n := len(parts); n > 4 {
			half := (n + 1) / 2
			return "<p align=\"center\">\n" + strings.Join(parts[:half], "\n") + "\n<br>\n" + strings.Join(parts[half:], "\n") + "\n</p>\n"
		}
		return "<p align=\"center\">\n" + strings.Join(parts, "\n") + "\n</p>\n"
	}
	var every []string
	for _, s := range links {
		every = append(every, s.key)
	}
	return map[string]string{
		"SOCIALS": row(every),
		"CONTACT": row([]string{"email", "linkedin", "github"}),
	}, files, nil
}

const (
	btnW = 188
	btnH = 52
	btnM = 5
)

func socialButton(s social, t theme, icons map[string]iconSVG) []byte {
	x := html.EscapeString
	color := readable(s.color, t)
	ow, oh := btnW+2*btnM, btnH+2*btnM
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" role="img" aria-label="%s">`, ow, oh, ow, oh, x(s.name+" — "+s.handle))
	fmt.Fprintf(&b, `<style>text{font-family:%s}</style><g transform="translate(%d %d)">`, fontStk, btnM, btnM)
	fmt.Fprintf(&b, `<rect x=".5" y=".5" width="%d" height="%d" rx="12" fill="%s" stroke="%s"/>`, btnW-1, btnH-1, t.bg, t.border)
	fmt.Fprintf(&b, `<rect x="10" y="10" width="32" height="32" rx="9" fill="%s" fill-opacity=".14"/>`, color)
	if ic, ok := icons[s.icon]; ok {
		fmt.Fprintf(&b, `<svg x="17" y="17" width="18" height="18" viewBox="0 0 %d %d" color="%s">%s</svg>`, ic.Width, ic.Height, color, ic.Body)
	} else {
		fmt.Fprintf(&b, `<circle cx="26" cy="26" r="5" fill="%s"/>`, color)
	}
	fmt.Fprintf(&b, `<text x="52" y="23" font-size="13" font-weight="700" fill="%s">%s</text>`, t.title, x(s.name))
	fmt.Fprintf(&b, `<text x="52" y="39" font-size="11" fill="%s">%s</text>`, t.muted, x(truncate(s.handle, 11, false, btnW-52-12)))
	b.WriteString("</g></svg>\n")
	return []byte(b.String())
}

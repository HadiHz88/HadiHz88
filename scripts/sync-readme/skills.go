package main

import (
	"errors"
	"fmt"
	"html"
	"path"
	"sort"
	"strings"
)

var skillGroups = []struct{ category, heading string }{
	{"language", "Languages"},
	{"frontend", "Frontend"},
	{"backend", "Backend"},
	{"databases", "Databases & ORMs"},
	{"devops", "Cloud & DevOps"},
	{"tools", "Tools"},
	{"other", "Other"},
}

type skillGroup struct {
	heading string
	tags    []Tag
}

func groupSkills(tags []Tag) []skillGroup {
	var featured []Tag
	for _, t := range tags {
		if t.Featured {
			featured = append(featured, t)
		}
	}
	sort.SliceStable(featured, func(i, j int) bool {
		if featured[i].Level != featured[j].Level {
			return featured[i].Level > featured[j].Level
		}
		return strings.ToLower(featured[i].Name) < strings.ToLower(featured[j].Name)
	})
	by := map[string][]Tag{}
	for _, t := range featured {
		by[t.Category] = append(by[t.Category], t)
	}
	var out []skillGroup
	for _, g := range skillGroups {
		if len(by[g.category]) > 0 {
			out = append(out, skillGroup{g.heading, by[g.category]})
		}
	}
	return out
}

// renderSkills returns the README block plus the banner files it references.
func renderSkills(tags []Tag, icons map[string]iconSVG, assetDir, site string) (string, map[string][]byte, error) {
	groups := groupSkills(tags)
	if len(groups) == 0 {
		return "", nil, errors.New("no featured skills")
	}
	dark := path.Join(assetDir, "skills-dark.svg")
	light := path.Join(assetDir, "skills-light.svg")
	files := map[string][]byte{
		dark:  banner(groups, darkTheme, icons),
		light: banner(groups, lightTheme, icons),
	}

	parts := make([]string, len(groups))
	for i, g := range groups {
		names := make([]string, len(g.tags))
		for j, t := range g.tags {
			names[j] = t.Name
		}
		parts[i] = g.heading + ": " + strings.Join(names, ", ")
	}
	alt := html.EscapeString("Tech stack — " + strings.Join(parts, " · "))
	href := html.EscapeString(strings.TrimRight(site, "/") + "/tags")
	body := fmt.Sprintf(
		"<p align=\"center\">\n<a href=\"%s\"><picture><source media=\"(prefers-color-scheme: dark)\" srcset=\"%s\"><img alt=\"%s\" src=\"%s\" width=\"100%%\"></picture></a>\n</p>\n",
		href, dark, alt, light)
	return body, files, nil
}

const (
	bannerW   = 880
	bannerPad = 28
	labelW    = 156
	chipH     = 34
	chipGap   = 8
)

type chip struct {
	tag  Tag
	x, y float64
	w    float64
}

func chipWidth(t Tag, icons map[string]iconSVG) float64 {
	if _, ok := icons[iconName(t)]; ok {
		return textWidth(t.Name, 13, false) + 50
	}
	return textWidth(t.Name, 13, false) + 42
}

func banner(groups []skillGroup, t theme, icons map[string]iconSVG) []byte {
	x := html.EscapeString
	left := float64(bannerPad + labelW)
	right := float64(bannerW - bannerPad)

	// Lay out first so the height is known before the SVG header is written.
	type row struct {
		heading string
		y       float64
		chips   []chip
	}
	var rows []row
	y := float64(bannerPad) + 4
	for i, g := range groups {
		if i > 0 {
			y += 16 + 1 + 16 // gap, divider, gap
		}
		r := row{heading: g.heading, y: y}
		cx := left
		for _, tag := range g.tags {
			w := chipWidth(tag, icons)
			if cx+w > right && cx > left {
				cx = left
				y += chipH + chipGap
			}
			r.chips = append(r.chips, chip{tag, cx, y, w})
			cx += w + chipGap
		}
		y += chipH
		rows = append(rows, r)
	}
	h := int(y) + bannerPad

	var names []string
	for _, g := range groups {
		for _, tag := range g.tags {
			names = append(names, tag.Name)
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" role="img" aria-labelledby="t d">`, bannerW, h, bannerW, h)
	fmt.Fprintf(&b, `<title id="t">Tech stack</title><desc id="d">%s</desc>`, x(strings.Join(names, ", ")))
	fmt.Fprintf(&b, `<style>text{font-family:%s}</style>`, fontStk)
	fmt.Fprintf(&b, `<defs><clipPath id="c"><rect width="%d" height="%d" rx="16"/></clipPath><linearGradient id="g" x1="0" x2="1"><stop offset="0" stop-color="%s"/><stop offset="1" stop-color="%s"/></linearGradient></defs>`, bannerW, h, t.status["completed"], t.accent)
	fmt.Fprintf(&b, `<g clip-path="url(#c)"><rect width="%d" height="%d" fill="%s"/><rect width="%d" height="3" fill="url(#g)"/></g>`, bannerW, h, t.bg, bannerW)
	fmt.Fprintf(&b, `<rect x=".5" y=".5" width="%d" height="%d" rx="15.5" fill="none" stroke="%s"/>`, bannerW-1, h-1, t.border)

	for i, r := range rows {
		if i > 0 {
			dy := r.y - 16.5
			fmt.Fprintf(&b, `<line x1="%d" y1="%.1f" x2="%d" y2="%.1f" stroke="%s" stroke-dasharray="2 5"/>`, bannerPad, dy, bannerW-bannerPad, dy, t.border)
		}
		fmt.Fprintf(&b, `<text x="%d" y="%.1f" font-size="11" font-weight="700" letter-spacing="1.2" fill="%s">%s</text>`, bannerPad, r.y+21, t.muted, x(strings.ToUpper(r.heading)))
		for _, c := range r.chips {
			color := t.muted
			if c.tag.Icon != nil && isHex(strings.TrimPrefix(c.tag.Icon.Color, "#")) {
				color = readable(c.tag.Icon.Color, t)
			}
			fmt.Fprintf(&b, `<rect x="%.1f" y="%.1f" width="%.1f" height="%d" rx="%d" fill="%s" fill-opacity=".08" stroke="%s" stroke-opacity=".4"/>`, c.x, c.y, c.w, chipH, chipH/2, color, color)
			tx := c.x + 32
			if ic, ok := icons[iconName(c.tag)]; ok {
				fmt.Fprintf(&b, `<svg x="%.1f" y="%.1f" width="18" height="18" viewBox="0 0 %d %d" color="%s">%s</svg>`, c.x+13, c.y+8, ic.Width, ic.Height, color, ic.Body)
				tx = c.x + 38
			} else {
				fmt.Fprintf(&b, `<circle cx="%.1f" cy="%.1f" r="4" fill="%s"/>`, c.x+19, c.y+17, color)
			}
			fmt.Fprintf(&b, `<text x="%.1f" y="%.1f" font-size="13" font-weight="500" fill="%s">%s</text>`, tx, c.y+21.5, t.title, x(c.tag.Name))
		}
	}
	b.WriteString("</svg>\n")
	return []byte(b.String())
}

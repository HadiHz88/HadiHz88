package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
)

type githubStats struct {
	Contributions int
	Stars         int
	Repos         int
	PullRequests  int
	Followers     int
	Languages     []language
}

type language struct {
	Name  string
	Color string
	Share float64
}

// Markup and notebooks dominate byte counts without saying much about the code written.
var skipLanguages = map[string]bool{"HTML": true, "CSS": true, "SCSS": true, "Jupyter Notebook": true}

const statsQuery = `query($login:String!){user(login:$login){
followers{totalCount}
contributionsCollection{totalPullRequestContributions contributionCalendar{totalContributions}}
repositories(first:100,ownerAffiliations:OWNER,isFork:false,privacy:PUBLIC){totalCount nodes{stargazerCount
languages(first:10,orderBy:{field:SIZE,direction:DESC}){edges{size node{name color}}}}}}}`

func fetchGitHubStats(ctx context.Context, token, login string) (githubStats, error) {
	payload, _ := json.Marshal(map[string]any{"query": statsQuery, "variables": map[string]string{"login": login}})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.github.com/graphql", bytes.NewReader(payload))
	if err != nil {
		return githubStats{}, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	res, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return githubStats{}, fmt.Errorf("github graphql: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return githubStats{}, fmt.Errorf("github graphql: %s", res.Status)
	}
	var out struct {
		Data struct {
			User *struct {
				Followers               struct{ TotalCount int }
				ContributionsCollection struct {
					TotalPullRequestContributions int
					ContributionCalendar          struct{ TotalContributions int }
				}
				Repositories struct {
					TotalCount int
					Nodes      []struct {
						StargazerCount int
						Languages      struct {
							Edges []struct {
								Size int
								Node struct{ Name, Color string }
							}
						}
					}
				}
			}
		}
		Errors []struct{ Message string }
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 8<<20)).Decode(&out); err != nil {
		return githubStats{}, fmt.Errorf("github graphql: decode: %w", err)
	}
	if len(out.Errors) > 0 {
		return githubStats{}, fmt.Errorf("github graphql: %s", out.Errors[0].Message)
	}
	u := out.Data.User
	if u == nil {
		return githubStats{}, fmt.Errorf("github graphql: user %q not found", login)
	}

	s := githubStats{
		Contributions: u.ContributionsCollection.ContributionCalendar.TotalContributions,
		Repos:         u.Repositories.TotalCount,
		PullRequests:  u.ContributionsCollection.TotalPullRequestContributions,
		Followers:     u.Followers.TotalCount,
	}
	sizes, colors := map[string]int{}, map[string]string{}
	total := 0
	for _, r := range u.Repositories.Nodes {
		s.Stars += r.StargazerCount
		for _, e := range r.Languages.Edges {
			if skipLanguages[e.Node.Name] {
				continue
			}
			sizes[e.Node.Name] += e.Size
			colors[e.Node.Name] = e.Node.Color
			total += e.Size
		}
	}
	for name, size := range sizes {
		s.Languages = append(s.Languages, language{name, colors[name], float64(size) / float64(total)})
	}
	sort.Slice(s.Languages, func(i, j int) bool {
		if s.Languages[i].Share != s.Languages[j].Share {
			return s.Languages[i].Share > s.Languages[j].Share
		}
		return s.Languages[i].Name < s.Languages[j].Name
	})
	if len(s.Languages) > 8 {
		s.Languages = s.Languages[:8]
	}
	return s, nil
}

func renderStats(s githubStats, assetDir, login string) (string, map[string][]byte, error) {
	if s.Repos == 0 {
		return "", nil, errors.New("no public repositories")
	}
	dark := path.Join(assetDir, "stats-dark.svg")
	light := path.Join(assetDir, "stats-light.svg")
	files := map[string][]byte{dark: statsBanner(s, darkTheme), light: statsBanner(s, lightTheme)}
	alt := fmt.Sprintf("GitHub stats — %s contributions in the last year, %s stars, %s public repositories",
		thousands(s.Contributions), thousands(s.Stars), thousands(s.Repos))
	body := fmt.Sprintf(
		"<p align=\"center\">\n<a href=\"https://github.com/%s?tab=repositories\"><picture><source media=\"(prefers-color-scheme: dark)\" srcset=\"%s\"><img alt=\"%s\" src=\"%s\" width=\"100%%\"></picture></a>\n</p>\n",
		html.EscapeString(login), dark, html.EscapeString(alt), light)
	return body, files, nil
}

func thousands(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

func statsBanner(s githubStats, t theme) []byte {
	x := html.EscapeString
	tiles := []struct {
		value int
		label string
	}{
		{s.Contributions, "Contributions · 1y"},
		{s.Stars, "Stars earned"},
		{s.Repos, "Public repos"},
		{s.PullRequests, "Pull requests · 1y"},
		{s.Followers, "Followers"},
	}
	const gap = 12.0
	inner := float64(bannerW - 2*bannerPad)
	tileW := (inner - gap*float64(len(tiles)-1)) / float64(len(tiles))
	const tileH = 78.0
	legendRows := (len(s.Languages) + 3) / 4
	h := bannerPad + 4 + int(tileH) + 30 + 14 + 12 + 10 + 14 + legendRows*22 + bannerPad - 8

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" role="img" aria-label="GitHub stats">`, bannerW, h, bannerW, h)
	fmt.Fprintf(&b, `<style>text{font-family:%s}</style>`, fontStk)
	fmt.Fprintf(&b, `<defs><clipPath id="c"><rect width="%d" height="%d" rx="16"/></clipPath><clipPath id="bar"><rect x="%d" width="%.1f" height="10" rx="5"/></clipPath><linearGradient id="g" x1="0" x2="1"><stop offset="0" stop-color="%s"/><stop offset="1" stop-color="%s"/></linearGradient></defs>`,
		bannerW, h, bannerPad, inner, t.status["active"], t.accent)
	fmt.Fprintf(&b, `<g clip-path="url(#c)"><rect width="%d" height="%d" fill="%s"/><rect width="%d" height="3" fill="url(#g)"/></g>`, bannerW, h, t.bg, bannerW)
	fmt.Fprintf(&b, `<rect x=".5" y=".5" width="%d" height="%d" rx="15.5" fill="none" stroke="%s"/>`, bannerW-1, h-1, t.border)

	y := float64(bannerPad + 4)
	for i, tile := range tiles {
		tx := float64(bannerPad) + float64(i)*(tileW+gap)
		fmt.Fprintf(&b, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.0f" rx="12" fill="%s" stroke="%s"/>`, tx, y, tileW, tileH, t.chip, t.border)
		fmt.Fprintf(&b, `<text x="%.1f" y="%.1f" font-size="26" font-weight="700" fill="%s">%s</text>`, tx+16, y+40, t.title, thousands(tile.value))
		fmt.Fprintf(&b, `<text x="%.1f" y="%.1f" font-size="10.5" font-weight="700" letter-spacing=".8" fill="%s">%s</text>`, tx+16, y+60, t.muted, x(strings.ToUpper(tile.label)))
	}
	y += tileH + 30

	fmt.Fprintf(&b, `<text x="%d" y="%.1f" font-size="11" font-weight="700" letter-spacing="1.2" fill="%s">TOP LANGUAGES</text>`, bannerPad, y+10, t.muted)
	y += 14 + 12
	fmt.Fprintf(&b, `<g transform="translate(0 %.1f)" clip-path="url(#bar)"><rect x="%d" width="%.1f" height="10" fill="%s"/>`, y, bannerPad, inner, t.pipOff)
	bx := float64(bannerPad)
	for _, l := range s.Languages {
		w := l.Share * inner
		fmt.Fprintf(&b, `<rect x="%.1f" width="%.1f" height="10" fill="%s"/>`, bx, w+0.5, langColor(l, t))
		bx += w
	}
	b.WriteString(`</g>`)
	y += 10 + 14

	colW := inner / 4
	for i, l := range s.Languages {
		lx := float64(bannerPad) + float64(i%4)*colW
		ly := y + float64(i/4)*22
		fmt.Fprintf(&b, `<circle cx="%.1f" cy="%.1f" r="4.5" fill="%s"/>`, lx+5, ly+8, langColor(l, t))
		fmt.Fprintf(&b, `<text x="%.1f" y="%.1f" font-size="12.5" fill="%s">%s <tspan fill="%s">%.1f%%</tspan></text>`, lx+16, ly+12.5, t.text, x(l.Name), t.muted, l.Share*100)
	}
	b.WriteString("</svg>\n")
	return []byte(b.String())
}

func langColor(l language, t theme) string {
	if isHex(strings.TrimPrefix(l.Color, "#")) {
		return l.Color
	}
	return t.muted
}

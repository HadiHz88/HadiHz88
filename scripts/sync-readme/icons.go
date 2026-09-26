package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// iconSVG is one Iconify icon: inner SVG markup plus its viewBox size.
type iconSVG struct {
	Body   string
	Width  int
	Height int
}

type iconifySet struct {
	Width   int `json:"width"`
	Height  int `json:"height"`
	Aliases map[string]struct {
		Parent string `json:"parent"`
	} `json:"aliases"`
	Icons map[string]struct {
		Body   string `json:"body"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	} `json:"icons"`
}

// fetchIcons resolves Iconify names ("simple-icons:astro") to inline SVG.
// Cards are <img> sources and cannot load anything external, so the paths
// must be embedded. A failed set is reported and its chips fall back to dots.
func fetchIcons(ctx context.Context, names []string) (map[string]iconSVG, error) {
	bySet := map[string][]string{}
	for _, n := range names {
		if prefix, name, ok := strings.Cut(n, ":"); ok && prefix != "" && name != "" {
			bySet[prefix] = append(bySet[prefix], name)
		}
	}
	prefixes := make([]string, 0, len(bySet))
	for p := range bySet {
		prefixes = append(prefixes, p)
	}
	sort.Strings(prefixes)

	hc := &http.Client{Timeout: 10 * time.Second}
	out := map[string]iconSVG{}
	var errs []string
	for _, prefix := range prefixes {
		set, err := fetchIconSet(ctx, hc, prefix, bySet[prefix])
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		for _, name := range bySet[prefix] {
			key := name
			if _, ok := set.Icons[key]; !ok {
				key = set.Aliases[name].Parent
			}
			ic, ok := set.Icons[key]
			if !ok || !safeIconBody(ic.Body) {
				continue
			}
			w, h := ic.Width, ic.Height
			if w == 0 {
				w = set.Width
			}
			if h == 0 {
				h = set.Height
			}
			if w == 0 || h == 0 {
				w, h = 16, 16
			}
			out[prefix+":"+name] = iconSVG{Body: ic.Body, Width: w, Height: h}
		}
	}
	if len(errs) > 0 {
		return out, fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return out, nil
}

func fetchIconSet(ctx context.Context, hc *http.Client, prefix string, names []string) (*iconifySet, error) {
	u := "https://api.iconify.design/" + url.PathEscape(prefix) + ".json?icons=" + url.QueryEscape(strings.Join(names, ","))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	res, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("iconify %s: %w", prefix, err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("iconify %s: %s", prefix, res.Status)
	}
	var set iconifySet
	if err := json.NewDecoder(io.LimitReader(res.Body, 4<<20)).Decode(&set); err != nil {
		return nil, fmt.Errorf("iconify %s: decode: %w", prefix, err)
	}
	return &set, nil
}

// safeIconBody keeps third-party markup to plain shapes before it is inlined.
func safeIconBody(body string) bool {
	l := strings.ToLower(body)
	for _, bad := range []string{"<script", "javascript:", " on", "<foreignobject", "href="} {
		if strings.Contains(l, bad) {
			return false
		}
	}
	return body != ""
}

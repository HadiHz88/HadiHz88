// Command sync-readme refreshes the CMS-driven blocks of the profile README.
//
// Each section is fetched concurrently and fails soft: on any error the old
// block stays, a GitHub Actions warning is printed, and the exit code stays 0.
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type result struct {
	body  string
	files map[string][]byte
	err   error
}

type section struct {
	marker string
	run    func(ctx context.Context) result
}

func main() {
	readme := flag.String("readme", "README.md", "README to update in place")
	assets := flag.String("assets", "assets/generated/projects", "directory for project cards, relative to the README")
	site := flag.String("site", "https://hadihz.me", "base URL project cards link to")
	flag.Parse()

	base := strings.TrimRight(os.Getenv("STRAPI_URL"), "/")
	token := os.Getenv("STRAPI_TOKEN")
	if base == "" || token == "" {
		warn("STRAPI_URL or STRAPI_TOKEN is not set; README left unchanged")
		return
	}

	doc, err := os.ReadFile(*readme)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	c := newClient(base, token)
	now := time.Now().UTC()
	assetDir := filepath.ToSlash(*assets)
	sections := []section{
		{"PROJECTS", func(ctx context.Context) result {
			items, err := fetchAll[Project](ctx, c, "/api/projects", query(
				"filters[featured][$eq]", "true",
				"populate[tags][fields][0]", "name",
				"populate[tags][fields][1]", "icon",
			))
			if err != nil {
				return result{err: err}
			}
			var names []string
			for _, p := range selectProjects(items) {
				for _, t := range p.Tags {
					if n := iconName(t); n != "" {
						names = append(names, n)
					}
				}
			}
			icons, err := fetchIcons(ctx, names)
			if err != nil {
				warn(fmt.Sprintf("PROJECTS: icons: %v; affected chips fall back to dots", err))
			}
			body, files, err := renderProjects(items, icons, assetDir, *site)
			return result{body: body, files: files, err: err}
		}},
		{"EXPERIENCE", func(ctx context.Context) result {
			items, err := fetchAll[Experience](ctx, c, "/api/experiences", query("filters[featured][$eq]", "true"))
			if err != nil {
				return result{err: err}
			}
			body, err := renderNow(items, now)
			return result{body: body, err: err}
		}},
		{"EDUCATION", func(ctx context.Context) result {
			items, err := fetchAll[Education](ctx, c, "/api/educations", query("filters[featured][$eq]", "true"))
			if err != nil {
				return result{err: err}
			}
			body, err := renderEducation(items, now)
			return result{body: body, err: err}
		}},
		{"SKILLS", func(ctx context.Context) result {
			items, err := fetchAll[Tag](ctx, c, "/api/tags", query("filters[isSkill][$eq]", "true"))
			if err != nil {
				return result{err: err}
			}
			body, err := renderSkills(items)
			return result{body: body, err: err}
		}},
	}

	ctx := context.Background()
	results := make([]result, len(sections))
	var wg sync.WaitGroup
	for i, s := range sections {
		wg.Add(1)
		go func(i int, s section) {
			defer wg.Done()
			results[i] = s.run(ctx)
		}(i, s)
	}
	wg.Wait()

	updated := 0
	for i, s := range sections {
		r := results[i]
		if r.err != nil {
			warn(fmt.Sprintf("%s: %v; keeping the previous block", s.marker, r.err))
			continue
		}
		next, err := replaceBlock(doc, s.marker, r.body)
		if err != nil {
			warn(fmt.Sprintf("%s: %v", s.marker, err))
			continue
		}
		if r.files != nil {
			if err := writeCards(filepath.Join(filepath.Dir(*readme), filepath.FromSlash(assetDir)), r.files); err != nil {
				warn(fmt.Sprintf("%s: writing cards: %v; keeping the previous block", s.marker, err))
				continue
			}
		}
		doc = next
		updated++
	}

	old, _ := os.ReadFile(*readme)
	if !bytes.Equal(old, doc) {
		if err := os.WriteFile(*readme, doc, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	fmt.Printf("sync-readme: %d/%d sections refreshed\n", updated, len(sections))
}

// writeCards writes the current cards and removes ones for projects that are
// no longer featured, so the directory mirrors the README.
func writeCards(dir string, files map[string][]byte) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	keep := map[string]bool{}
	for name, data := range files {
		base := path.Base(name)
		keep[base] = true
		if err := os.WriteFile(filepath.Join(dir, base), data, 0o644); err != nil {
			return err
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".svg") && !keep[e.Name()] {
			if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

// warn prints a GitHub Actions warning annotation; outside Actions it is just a line.
func warn(msg string) {
	msg = strings.NewReplacer("%", "%25", "\r", "%0D", "\n", "%0A").Replace(msg)
	fmt.Printf("::warning title=sync-readme::%s\n", msg)
}

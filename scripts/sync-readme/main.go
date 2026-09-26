// Command sync-readme refreshes the generated blocks of the profile README:
// CMS content from Strapi and activity stats from GitHub.
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
	"sort"
	"strings"
	"sync"
	"time"
)

type result struct {
	blocks map[string]string // marker name -> block body
	dir    string            // where files are written; the renderer's paths share this prefix
	files  map[string][]byte
	err    error
}

type section struct {
	name string
	run  func(ctx context.Context) result
}

func single(marker, body string) map[string]string { return map[string]string{marker: body} }

func main() {
	readme := flag.String("readme", "README.md", "README to update in place")
	assets := flag.String("assets", "assets/generated", "directory for generated SVGs, relative to the README")
	site := flag.String("site", "https://hadihz.me", "base URL the cards and banner link to")
	flag.Parse()

	doc, err := os.ReadFile(*readme)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	now := time.Now().UTC()
	dir := func(name string) string { return path.Join(filepath.ToSlash(*assets), name) }
	var sections []section

	base := strings.TrimRight(os.Getenv("STRAPI_URL"), "/")
	token := os.Getenv("STRAPI_TOKEN")
	if base == "" || token == "" {
		warn("STRAPI_URL or STRAPI_TOKEN is not set; CMS sections left unchanged")
	} else {
		c := newClient(base, token)
		sections = append(sections,
			section{"PROJECTS", func(ctx context.Context) result {
				items, err := fetchAll[Project](ctx, c, "/api/projects", query(
					"filters[featured][$eq]", "true",
					"populate[tags][fields][0]", "name",
					"populate[tags][fields][1]", "icon",
				))
				if err != nil {
					return result{err: err}
				}
				var tags []Tag
				for _, p := range selectProjects(items) {
					tags = append(tags, p.Tags...)
				}
				body, files, err := renderProjects(items, iconsFor(ctx, "PROJECTS", tags), dir("projects"), *site)
				return result{blocks: single("PROJECTS", body), dir: dir("projects"), files: files, err: err}
			}},
			section{"EXPERIENCE", func(ctx context.Context) result {
				items, err := fetchAll[Experience](ctx, c, "/api/experiences", query("filters[featured][$eq]", "true"))
				if err != nil {
					return result{err: err}
				}
				body, err := renderNow(items, now)
				return result{blocks: single("EXPERIENCE", body), err: err}
			}},
			section{"EDUCATION", func(ctx context.Context) result {
				items, err := fetchAll[Education](ctx, c, "/api/educations", query("filters[featured][$eq]", "true"))
				if err != nil {
					return result{err: err}
				}
				body, err := renderEducation(items, now)
				return result{blocks: single("EDUCATION", body), err: err}
			}},
			section{"SKILLS", func(ctx context.Context) result {
				items, err := fetchAll[Tag](ctx, c, "/api/tags", query(
					"filters[isSkill][$eq]", "true",
					"filters[featured][$eq]", "true",
				))
				if err != nil {
					return result{err: err}
				}
				body, files, err := renderSkills(items, iconsFor(ctx, "SKILLS", items), dir("skills"), *site)
				return result{blocks: single("SKILLS", body), dir: dir("skills"), files: files, err: err}
			}},
			section{"PROFILE", func(ctx context.Context) result {
				p, err := fetchProfile(ctx, c)
				if err != nil {
					return result{err: err}
				}
				links, notes := socialLinks(p)
				for _, n := range notes {
					warn("PROFILE: " + n + "; button skipped")
				}
				var names []string
				for _, l := range links {
					names = append(names, l.icon)
				}
				icons, err := fetchIcons(ctx, names)
				if err != nil {
					warn(fmt.Sprintf("PROFILE: icons: %v; affected buttons fall back to dots", err))
				}
				blocks, files, err := renderSocials(links, icons, dir("profile"))
				if err != nil {
					return result{err: err}
				}
				if about := renderAbout(p); about != "" {
					blocks["ABOUT"] = about
				} else {
					warn("PROFILE: description is empty; keeping the previous About block")
				}
				if lines := typingLines(p); len(lines) > 0 {
					body, typing := renderTyping(lines, dir("profile"))
					blocks["TYPING"] = body
					for k, v := range typing {
						files[k] = v
					}
				} else {
					warn("PROFILE: no headline, position or location; keeping the previous typing banner")
				}
				return result{blocks: blocks, dir: dir("profile"), files: files}
			}},
		)
	}

	ghToken := os.Getenv("GITHUB_TOKEN")
	login := os.Getenv("GITHUB_REPOSITORY_OWNER")
	if login == "" {
		login = "HadiHz88"
	}
	if ghToken == "" {
		warn("GITHUB_TOKEN is not set; STATS left unchanged")
	} else {
		sections = append(sections, section{"STATS", func(ctx context.Context) result {
			s, err := fetchGitHubStats(ctx, ghToken, login)
			if err != nil {
				return result{err: err}
			}
			body, files, err := renderStats(s, dir("stats"), login)
			return result{blocks: single("STATS", body), dir: dir("stats"), files: files, err: err}
		}})
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
			warn(fmt.Sprintf("%s: %v; keeping the previous block", s.name, r.err))
			continue
		}
		next, err := replaceAll(doc, r.blocks)
		if err != nil {
			warn(fmt.Sprintf("%s: %v", s.name, err))
			continue
		}
		if r.files != nil {
			if err := writeCards(filepath.Join(filepath.Dir(*readme), filepath.FromSlash(r.dir)), r.files); err != nil {
				warn(fmt.Sprintf("%s: writing SVGs: %v; keeping the previous block", s.name, err))
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

// replaceAll applies every block of one section, or none of them.
func replaceAll(doc []byte, blocks map[string]string) ([]byte, error) {
	names := make([]string, 0, len(blocks))
	for n := range blocks {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		next, err := replaceBlock(doc, n, blocks[n])
		if err != nil {
			return nil, err
		}
		doc = next
	}
	return doc, nil
}

// iconsFor resolves the tags' icons; failures only cost the icons, not the section.
func iconsFor(ctx context.Context, marker string, tags []Tag) map[string]iconSVG {
	var names []string
	for _, t := range tags {
		if n := iconName(t); n != "" {
			names = append(names, n)
		}
	}
	icons, err := fetchIcons(ctx, names)
	if err != nil {
		warn(fmt.Sprintf("%s: icons: %v; affected chips fall back to dots", marker, err))
	}
	return icons
}

// writeCards writes the current SVGs and removes stale ones, so the directory
// mirrors the README.
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

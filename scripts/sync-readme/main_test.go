package main

import (
	"strings"
	"testing"
	"time"
)

func TestReplaceBlockPreservesSurroundingBytes(t *testing.T) {
	doc := "head\r\n  <!-- X:START -->\nold\n<!-- X:END -->\ttail  \n"
	got, err := replaceBlock([]byte(doc), "X", "\nnew\n")
	if err != nil {
		t.Fatal(err)
	}
	want := "head\r\n  <!-- X:START -->\nnew\n<!-- X:END -->\ttail  \n"
	if string(got) != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestReplaceBlockMissingOrReversedMarkers(t *testing.T) {
	for _, doc := range []string{"nothing", "<!-- X:START -->", "<!-- X:END --><!-- X:START -->"} {
		if _, err := replaceBlock([]byte(doc), "X", "b"); err == nil {
			t.Errorf("%q: expected error", doc)
		}
	}
}

func TestMarkdownEscape(t *testing.T) {
	got := md(" a|b [c](d) <x> \\ ")
	want := `a\|b \[c\](d) &lt;x&gt; \\`
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSafeURLRejectsNonHTTP(t *testing.T) {
	if safeURL("javascript:alert(1)") != "" || safeURL("/relative") != "" {
		t.Fatal("non-http URL accepted")
	}
	if got := safeURL("https://x.dev/a_(b)"); got != "https://x.dev/a_%28b%29" {
		t.Fatalf("got %q", got)
	}
}

func TestRenderSkillsKeepsFeaturedGroupedInOrder(t *testing.T) {
	tags := []Tag{
		{Name: "Hidden", Category: "language", Featured: false, Level: 99},
		{Name: "React", Category: "frontend", Featured: true, Level: 90},
		{Name: "TypeScript", Category: "language", Featured: true, Level: 90},
		{Name: "C#", Category: "language", Featured: true, Level: 70},
	}
	body, files, err := renderSkills(tags, nil, "assets/generated/skills", "https://x.dev")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(body, "Hidden") || !strings.Contains(body, "Languages: TypeScript, C# · Frontend: React") {
		t.Fatalf("got %s", body)
	}
	if len(files) != 2 || files["assets/generated/skills/skills-dark.svg"] == nil {
		t.Fatalf("files: %v", len(files))
	}
	if !strings.Contains(string(files["assets/generated/skills/skills-dark.svg"]), "C#") {
		t.Fatal("banner missing skill")
	}
}

func TestReadableSwapsInvisibleColours(t *testing.T) {
	if readable("#000000", darkTheme) != darkTheme.text || readable("#000000", lightTheme) != "#000000" {
		t.Fatal("black handling wrong")
	}
	if readable("#ffffff", lightTheme) != lightTheme.text {
		t.Fatal("white on light should swap")
	}
}

func TestRenderNowKeepsOnlyInProgress(t *testing.T) {
	now := time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC)
	got, err := renderNow([]Experience{
		{Title: "Old", Featured: true, StartDate: "2025-01-01", EndDate: "2025-06-01"},
		{Title: "Hidden", Featured: false, StartDate: "2026-01-01"},
		{Title: "Lab", Featured: true, StartDate: "2025-02-01", EmploymentType: "part-time"},
		{Title: "Job", Company: "Co", URL: "https://co.dev", Featured: true, StartDate: "2026-06-01", EmploymentType: "full-time"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "Old") || strings.Contains(got, "Hidden") {
		t.Fatalf("ended or unfeatured role rendered:\n%s", got)
	}
	if strings.Index(got, "Job") > strings.Index(got, "Lab") {
		t.Fatalf("newest role should come first:\n%s", got)
	}
	if !strings.Contains(got, "[**Co**](https://co.dev)") || !strings.Contains(got, "since Jun 2026") {
		t.Fatalf("got:\n%s", got)
	}
}

func TestSelectProjectsOrdersActiveFirstAndCaps(t *testing.T) {
	var items []Project
	for i := 0; i < 8; i++ {
		items = append(items, Project{Title: string(rune('A' + i)), Featured: true, Status: "completed", StartDate: "2025-01-0" + string(rune('1'+i))})
	}
	items = append(items, Project{Title: "Z", Featured: true, Status: "active"}, Project{Title: "N", Featured: false, Status: "active"})
	got := selectProjects(items)
	if len(got) != maxProjects || got[0].Title != "Z" || got[1].Title != "H" {
		t.Fatalf("got %+v", got)
	}
}

func TestSlugifyMatchesSite(t *testing.T) {
	cases := map[string]string{
		"hadihz.me":                           "hadihz-me",
		"Bikhedemtak — Local Service Network": "bikhedemtak-local-service-network",
		"  CloudQ ":                           "cloudq",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestWrapLimitsLines(t *testing.T) {
	lines := wrap(strings.Repeat("word ", 200), 13.5, 392, 3)
	if len(lines) != 3 || !strings.HasSuffix(lines[2], "…") {
		t.Fatalf("got %q", lines)
	}
	for _, l := range lines {
		if textWidth(l, 13.5, false) > 392 {
			t.Fatalf("line too wide: %q", l)
		}
	}
}

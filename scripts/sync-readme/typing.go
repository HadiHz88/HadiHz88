package main

import (
	"fmt"
	"html"
	"path"
	"strings"
	"unicode/utf8"
)

var htmlEscaper = strings.NewReplacer("<", "&lt;", ">", "&gt;")

// renderAbout passes the CMS rich text through as Markdown (it is authored as
// Markdown) with raw HTML neutralised.
func renderAbout(p Profile) string {
	return htmlEscaper.Replace(strings.TrimSpace(p.Description))
}

func typingLines(p Profile) []string {
	var lines []string
	for _, l := range []string{p.Headline, p.Position} {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	if place := strings.Trim(strings.TrimSpace(p.City)+", "+strings.TrimSpace(p.Country), ", "); place != "" {
		lines = append(lines, "Based in "+place)
	}
	return lines
}

func renderTyping(lines []string, assetDir string) (string, map[string][]byte) {
	dark := path.Join(assetDir, "typing-dark.svg")
	light := path.Join(assetDir, "typing-light.svg")
	files := map[string][]byte{
		dark:  typingSVG(lines, "#36BCF7"),
		light: typingSVG(lines, "#0969DA"),
	}
	body := fmt.Sprintf(
		"<picture><source media=\"(prefers-color-scheme: dark)\" srcset=\"%s\"><img alt=\"%s\" src=\"%s\"></picture>\n",
		dark, html.EscapeString(strings.Join(lines, " · ")), light)
	return body, files
}

const (
	typeW      = 650
	typeH      = 50
	typeSize   = 22.0
	typeCharW  = typeSize * 0.6 // monospace advance of every font in the stack below
	typeChar   = 0.07           // seconds per typed character
	typeDelete = 0.03           // seconds per deleted character
	typeHold   = 1.8
	typeGap    = 0.4
)

// typingSVG types and deletes each line in turn. SMIL rather than CSS: GitHub
// serves this as an <img>, where SMIL runs everywhere and discrete steps give
// a per-character reveal. Consolas is left out of the font stack: its 0.55em
// advance would let the cursor run ahead of the text.
func typingSVG(lines []string, color string) []byte {
	type step struct{ t, w float64 }
	type track struct {
		x0    float64
		steps []step
	}
	var tracks []track
	t := 0.0
	for _, l := range lines {
		n := utf8.RuneCountInString(l)
		tr := track{x0: (typeW - float64(n)*typeCharW) / 2}
		tr.steps = append(tr.steps, step{0, 0})
		for k := 1; k <= n; k++ {
			tr.steps = append(tr.steps, step{t + float64(k)*typeChar, float64(k) * typeCharW})
		}
		t += float64(n)*typeChar + typeHold
		for k := n - 1; k >= 0; k-- {
			t += typeDelete
			tr.steps = append(tr.steps, step{t, float64(k) * typeCharW})
		}
		t += typeGap
		tracks = append(tracks, tr)
	}
	total := t

	anim := func(attr string, steps []step, offset float64) string {
		var vals, times []string
		for _, s := range steps {
			vals = append(vals, fmt.Sprintf("%.1f", s.w+offset))
			times = append(times, fmt.Sprintf("%.4f", s.t/total))
		}
		return fmt.Sprintf(`<animate attributeName="%s" values="%s" keyTimes="%s" dur="%.2fs" calcMode="discrete" repeatCount="indefinite"/>`,
			attr, strings.Join(vals, ";"), strings.Join(times, ";"), total)
	}

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" role="img" aria-label="%s">`,
		typeW, typeH, typeW, typeH, html.EscapeString(strings.Join(lines, " · ")))
	b.WriteString(`<defs>`)
	for i, tr := range tracks {
		fmt.Fprintf(&b, `<clipPath id="l%d"><rect x="%.1f" y="0" width="0" height="%d">%s</rect></clipPath>`, i, tr.x0, typeH, anim("width", tr.steps, 0))
	}
	b.WriteString(`</defs>`)
	for i, l := range lines {
		fmt.Fprintf(&b, `<text x="%.1f" y="33" clip-path="url(#l%d)" font-family="'Fira Code',ui-monospace,SFMono-Regular,Menlo,'DejaVu Sans Mono','Liberation Mono','Courier New',monospace" font-size="%.0f" fill="%s" xml:space="preserve">%s</text>`,
			tracks[i].x0, i, typeSize, color, html.EscapeString(l))
	}

	// One cursor follows whichever line is active; lines never overlap in time.
	var cursor []step
	for _, tr := range tracks {
		for _, s := range tr.steps[1:] {
			cursor = append(cursor, step{s.t, tr.x0 + s.w})
		}
	}
	cursor = append([]step{{0, tracks[0].x0}}, cursor...)
	fmt.Fprintf(&b, `<rect x="%.1f" y="12" width="2" height="26" fill="%s">%s<animate attributeName="opacity" values="1;0" dur="1s" calcMode="discrete" repeatCount="indefinite"/></rect>`,
		tracks[0].x0, color, anim("x", cursor, 1))
	b.WriteString("</svg>\n")
	return []byte(b.String())
}

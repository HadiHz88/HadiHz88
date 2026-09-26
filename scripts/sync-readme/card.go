package main

import (
	"fmt"
	"html"
	"strconv"
	"strings"
	"unicode"
)

type theme struct {
	bg, border, title, text, muted, chip, pipOff, accent string
	status                                               map[string]string
	dark                                                 bool
}

// Tokyo Night in dark mode, to sit next to the tokyonight stats cards.
var darkTheme = theme{
	dark: true, bg: "#1a1b27", border: "#2a2e45", title: "#c0caf5", text: "#a9b1d6",
	muted: "#737aa2", chip: "#24283b", pipOff: "#3b4261", accent: "#bb9af7",
	status: map[string]string{
		"active": "#9ece6a", "planned": "#e0af68", "completed": "#7aa2f7",
		"archived": "#737aa2", "canceled": "#f7768e",
	},
}

var lightTheme = theme{
	bg: "#ffffff", border: "#d0d7de", title: "#1f2328", text: "#424a53",
	muted: "#6e7781", chip: "#f6f8fa", pipOff: "#d0d7de", accent: "#8250df",
	status: map[string]string{
		"active": "#1a7f37", "planned": "#9a6700", "completed": "#0969da",
		"archived": "#6e7781", "canceled": "#cf222e",
	},
}

var difficultyPips = map[string]int{"easy": 1, "medium": 2, "hard": 3, "advanced": 4}

const (
	cardW = 440
	cardH = 200
	padX  = 24
	// Transparent margin baked into each card: GitHub strips CSS, so this is
	// the only way to put a gap between cards laid out side by side.
	cardM   = 10
	fontStk = `-apple-system,BlinkMacSystemFont,'Segoe UI','Noto Sans',Helvetica,Arial,sans-serif`
)

func card(p Project, t theme, icons map[string]iconSVG) []byte {
	x := html.EscapeString
	sc := t.status[p.Status]
	if sc == "" {
		sc = t.muted
	}
	var b strings.Builder
	ow, oh := cardW+2*cardM, cardH+2*cardM
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" role="img" aria-labelledby="t d">`, ow, oh, ow, oh)
	fmt.Fprintf(&b, `<title id="t">%s</title><desc id="d">%s</desc>`, x(p.Title), x(p.Summary))
	fmt.Fprintf(&b, `<style>
text{font-family:%s}
.live{animation:live 2s ease-in-out infinite}
@keyframes live{50%%{opacity:.25}}
@media (prefers-reduced-motion:reduce){.live{animation:none}}
</style>`, fontStk)
	fmt.Fprintf(&b, `<g transform="translate(%d %d)">`, cardM, cardM)
	fmt.Fprintf(&b, `<defs><clipPath id="c"><rect width="%d" height="%d" rx="14"/></clipPath><linearGradient id="g" x1="0" x2="1"><stop offset="0" stop-color="%s"/><stop offset="1" stop-color="%s"/></linearGradient></defs>`, cardW, cardH, sc, t.accent)
	fmt.Fprintf(&b, `<g clip-path="url(#c)"><rect width="%d" height="%d" fill="%s"/><rect width="%d" height="3" fill="url(#g)"/></g>`, cardW, cardH, t.bg, cardW)
	fmt.Fprintf(&b, `<rect x=".5" y=".5" width="%d" height="%d" rx="13.5" fill="none" stroke="%s"/>`, cardW-1, cardH-1, t.border)

	// Status pill, top right; the title gets whatever width is left.
	status := strings.ToUpper(p.Status)
	pillW := textWidth(status, 10.5, true) + float64(len(status))*0.6 + 28
	pillX := float64(cardW-padX) - pillW
	fmt.Fprintf(&b, `<rect x="%.1f" y="24" width="%.1f" height="22" rx="11" fill="%s" fill-opacity=".12" stroke="%s" stroke-opacity=".45"/>`, pillX, pillW, sc, sc)
	dotClass := ""
	if p.Status == "active" {
		dotClass = ` class="live"`
	}
	fmt.Fprintf(&b, `<circle%s cx="%.1f" cy="35" r="3.5" fill="%s"/>`, dotClass, pillX+12, sc)
	fmt.Fprintf(&b, `<text x="%.1f" y="38.8" font-size="10.5" font-weight="700" letter-spacing=".6" fill="%s">%s</text>`, pillX+20, sc, x(status))

	titleMax := pillX - padX - 12
	fmt.Fprintf(&b, `<text x="%d" y="42" font-size="19" font-weight="700" fill="%s">%s</text>`, padX, t.title, x(truncate(p.Title, 19, true, titleMax)))

	// Type · difficulty pips.
	typ := title(p.Type)
	fmt.Fprintf(&b, `<text x="%d" y="68" font-size="12.5" fill="%s">%s</text>`, padX, t.muted, x(typ))
	px := float64(padX) + textWidth(typ, 12.5, false) + 12
	fmt.Fprintf(&b, `<text x="%.1f" y="68" font-size="12.5" fill="%s">·</text>`, px-8, t.muted)
	on := difficultyPips[p.Difficulty]
	for i := 0; i < 4; i++ {
		fill := t.pipOff
		if i < on {
			fill = t.accent
		}
		fmt.Fprintf(&b, `<rect x="%.1f" y="60" width="8" height="8" rx="2" fill="%s"/>`, px+float64(i)*11, fill)
	}
	fmt.Fprintf(&b, `<text x="%.1f" y="68" font-size="12.5" fill="%s">%s</text>`, px+50, t.muted, x(p.Difficulty))

	// Summary.
	for i, line := range wrap(p.Summary, 13.5, cardW-2*padX, 3) {
		fmt.Fprintf(&b, `<text x="%d" y="%d" font-size="13.5" fill="%s">%s</text>`, padX, 100+i*20, t.text, x(line))
	}

	// Tag chips; whatever does not fit collapses into "+N".
	cx := float64(padX)
	limit := float64(cardW - padX)
	for i, tag := range p.Tags {
		w := textWidth(tag.Name, 12, false) + 36
		rest := len(p.Tags) - i - 1
		reserve := 0.0
		if rest > 0 {
			reserve = 44
		}
		if cx+w > limit-reserve {
			more := fmt.Sprintf("+%d", len(p.Tags)-i)
			mw := textWidth(more, 12, true) + 20
			fmt.Fprintf(&b, `<rect x="%.1f" y="156" width="%.1f" height="24" rx="12" fill="%s" stroke="%s"/>`, cx, mw, t.chip, t.border)
			fmt.Fprintf(&b, `<text x="%.1f" y="172" font-size="12" font-weight="600" fill="%s">%s</text>`, cx+10, t.muted, more)
			break
		}
		color := t.muted
		if tag.Icon != nil && isHex(strings.TrimPrefix(tag.Icon.Color, "#")) {
			color = readable(tag.Icon.Color, t)
		}
		fmt.Fprintf(&b, `<rect x="%.1f" y="156" width="%.1f" height="24" rx="12" fill="%s" stroke="%s"/>`, cx, w, t.chip, t.border)
		if ic, ok := icons[iconName(tag)]; ok {
			fmt.Fprintf(&b, `<svg x="%.1f" y="161" width="14" height="14" viewBox="0 0 %d %d" color="%s">%s</svg>`, cx+9, ic.Width, ic.Height, color, ic.Body)
		} else {
			fmt.Fprintf(&b, `<circle cx="%.1f" cy="168" r="4" fill="%s"/>`, cx+16, color)
		}
		fmt.Fprintf(&b, `<text x="%.1f" y="172" font-size="12" fill="%s">%s</text>`, cx+28, t.text, x(tag.Name))
		cx += w + 6
	}

	b.WriteString(`</g></svg>`)
	b.WriteString("\n")
	return []byte(b.String())
}

func iconName(tag Tag) string {
	if tag.Icon == nil {
		return ""
	}
	return tag.Icon.Name
}

// readable swaps brand colours that vanish on the card background (black
// logos on dark, near-white on light) for the theme's text colour.
func readable(hex string, t theme) string {
	v, err := strconv.ParseUint(strings.TrimPrefix(hex, "#"), 16, 32)
	if err != nil {
		return t.text
	}
	r, g, b := float64(v>>16&0xff), float64(v>>8&0xff), float64(v&0xff)
	lum := 0.299*r + 0.587*g + 0.114*b
	if (t.dark && lum < 70) || (!t.dark && lum > 200) {
		return t.text
	}
	return hex
}

// textWidth estimates rendered width from per-glyph classes; SVG in an <img>
// cannot measure text, and system sans fonts are close enough to these ratios.
func textWidth(s string, size float64, bold bool) float64 {
	w := 0.0
	for _, r := range s {
		switch {
		case r == ' ':
			w += 0.28
		case strings.ContainsRune("iljtfI.,:;'!|()[]", r):
			w += 0.3
		case strings.ContainsRune("mwMW@—", r):
			w += 0.86
		case unicode.IsUpper(r):
			w += 0.66
		case unicode.IsDigit(r):
			w += 0.56
		case r > unicode.MaxASCII:
			w += 0.6
		default:
			w += 0.53
		}
	}
	if bold {
		w *= 1.07
	}
	return w * size
}

func truncate(s string, size float64, bold bool, max float64) string {
	if textWidth(s, size, bold) <= max {
		return s
	}
	r := []rune(s)
	for len(r) > 0 && textWidth(string(r)+"…", size, bold) > max {
		r = r[:len(r)-1]
	}
	return strings.TrimRight(string(r), " ") + "…"
}

func wrap(s string, size float64, max float64, lines int) []string {
	var out []string
	cur := ""
	words := strings.Fields(s)
	for i, w := range words {
		next := strings.TrimSpace(cur + " " + w)
		if textWidth(next, size, false) <= max || cur == "" {
			cur = next
			continue
		}
		if len(out) == lines-1 {
			out = append(out, truncate(strings.Join(append([]string{cur}, words[i:]...), " "), size, false, max))
			return out
		}
		out = append(out, cur)
		cur = w
	}
	if cur != "" {
		out = append(out, truncate(cur, size, false, max))
	}
	return out
}

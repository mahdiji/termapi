package nav

import (
	"fmt"
	"strings"
	"time"

	"api/internal/ui/styles"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// NavbarWidth menyimpan lebar terakhir navbar agar konten bisa di-justify.
var NavbarWidth = 0

// ActiveEnv dan GlobalScope diisi oleh AppModel sebelum render.
var (
	ActiveEnv   string
	GlobalScope string
)

// NavbarContent merender bar atas: brand badge solid di kiri, env/scope/jam
// di kanan. Brand pakai latar aksen penuh (gaya lazygit/k9s) agar langsung
// terlihat sebagai identitas aplikasi, bukan teks biasa.
func NavbarContent() string {
	brand := lipgloss.NewStyle().
		Foreground(styles.ColBackground).
		Background(styles.ColAccent).
		Bold(true).
		Padding(0, 1).
		Render("TermAPI")

	left := brand

	if NavbarWidth == 0 {
		return lipgloss.NewStyle().
			Background(styles.ColBarBg).
			Render(left)
	}

	// Kanan: env badge + scope + jam
	rightParts := []string{}
	if ActiveEnv != "" {
		dot := lipgloss.NewStyle().Foreground(styles.ColSuccess).Render(styles.IconDot)
		env := lipgloss.NewStyle().Foreground(styles.ColTextBright).Bold(true).Render(" " + ActiveEnv)
		rightParts = append(rightParts, dot+env)
	}
	if GlobalScope != "" {
		scope := lipgloss.NewStyle().
			Foreground(styles.ColTextMuted).
			Background(styles.ColSurface).
			Padding(0, 1).
			Render(scopeBadge(GlobalScope))
		rightParts = append(rightParts, scope)
	}
	clock := lipgloss.NewStyle().Foreground(styles.ColTextDim).Render(time.Now().Format("15:04"))
	rightParts = append(rightParts, clock)

	sep := lipgloss.NewStyle().Foreground(styles.ColBorder).Render(" │ ")
	right := ""
	for i, p := range rightParts {
		if i > 0 {
			right += sep
		}
		right += p
	}
	right += " "

	gap := max(NavbarWidth-lipgloss.Width(left)-lipgloss.Width(right), 0)
	bar := left + lipgloss.NewStyle().Width(gap).Render("") + right

	return lipgloss.NewStyle().
		Background(styles.ColBarBg).
		MaxWidth(NavbarWidth).
		Render(bar)
}

// scopeBadge merangkum scope multi-baris menjadi satu badge pendek agar
// navbar tidak rusak ketika ada lebih dari satu origin.
func scopeBadge(scope string) string {
	origins := make([]string, 0, 2)
	for _, line := range strings.Split(scope, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			origins = append(origins, line)
		}
	}
	switch len(origins) {
	case 0:
		return ""
	case 1:
		return ansi.Truncate(origins[0], 32, "…")
	default:
		return ansi.Truncate(fmt.Sprintf("%s +%d", origins[0], len(origins)-1), 32, "…")
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

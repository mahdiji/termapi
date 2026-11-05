package requestinput

import (
	"strings"

	"api/internal/ui/styles"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// OpenFind focuses the in-response text search field.
func (p *ResponsePanel) OpenFind() tea.Cmd {
	if p == nil {
		return nil
	}
	p.FindFocused = true
	p.recomputeMatches()
	return p.Find.Focus()
}

// CloseFind clears the query and returns the toolbar to the gjson filter.
func (p *ResponsePanel) CloseFind() {
	if p == nil {
		return
	}
	p.FindFocused = false
	p.Find.Blur()
	p.Find.SetValue("")
	p.Matches = nil
	p.MatchIdx = 0
}

// UpdateFind routes a message to the find input and refreshes the matches.
func (p *ResponsePanel) UpdateFind(msg tea.Msg) tea.Cmd {
	if p == nil {
		return nil
	}
	var cmd tea.Cmd
	p.Find, cmd = p.Find.Update(msg)
	p.recomputeMatches()
	return cmd
}

// FindActive reports whether a non-empty query is present.
func (p *ResponsePanel) FindActive() bool {
	return p != nil && strings.TrimSpace(p.Find.Value()) != ""
}

func (p *ResponsePanel) recomputeMatches() {
	if p == nil {
		return
	}
	p.Matches = nil
	query := strings.ToLower(strings.TrimSpace(p.Find.Value()))
	if query == "" {
		p.MatchIdx = 0
		return
	}
	for i, line := range p.plainLines() {
		if strings.Contains(strings.ToLower(line), query) {
			p.Matches = append(p.Matches, i)
		}
	}
	if len(p.Matches) == 0 {
		p.MatchIdx = 0
		return
	}
	if p.MatchIdx >= len(p.Matches) {
		p.MatchIdx = 0
	}
}

// NextMatch jumps to the next/previous matching line.
func (p *ResponsePanel) NextMatch(delta int) {
	if p == nil || len(p.Matches) == 0 {
		return
	}
	p.MatchIdx = (p.MatchIdx + delta + len(p.Matches)) % len(p.Matches)
	p.jumpToMatch()
}

func (p *ResponsePanel) jumpToMatch() {
	if len(p.Matches) == 0 {
		return
	}
	line := p.Matches[p.MatchIdx]
	height := max(p.vp.Height(), 1)
	total := p.vp.TotalLineCount()
	offset := line - height/2
	if offset > total-height {
		offset = total - height
	}
	if offset < 0 {
		offset = 0
	}
	p.vp.SetYOffset(offset)
	p.ScrollOffset = p.vp.YOffset()
}

func (p *ResponsePanel) activeMatchLine() int {
	if len(p.Matches) == 0 || p.MatchIdx < 0 || p.MatchIdx >= len(p.Matches) {
		return -1
	}
	return p.Matches[p.MatchIdx]
}

// highlightLine wraps every case-insensitive occurrence of the query. The
// active match uses a brighter background.
func (p *ResponsePanel) highlightLine(line string, active bool) string {
	query := p.Find.Value()
	if query == "" {
		return styles.Text().Render(line)
	}
	matchStyle := lipgloss.NewStyle().
		Foreground(styles.ColBackground).
		Background(styles.ColAccent).
		Bold(true)
	if active {
		matchStyle = lipgloss.NewStyle().
			Foreground(styles.ColBackground).
			Background(styles.ColAccentHi).
			Bold(true)
	}

	lowerLine := strings.ToLower(line)
	lowerQuery := strings.ToLower(query)
	var b strings.Builder
	last := 0
	for {
		idx := strings.Index(lowerLine[last:], lowerQuery)
		if idx < 0 {
			break
		}
		idx += last
		if idx > last {
			b.WriteString(styles.Text().Render(line[last:idx]))
		}
		b.WriteString(matchStyle.Render(line[idx : idx+len(query)]))
		last = idx + len(query)
	}
	b.WriteString(styles.Text().Render(line[last:]))
	return b.String()
}

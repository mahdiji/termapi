package requestinput

import (
	"fmt"

	"api/internal/ui/styles"

	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

// View merender bar request: [METHOD] [URL input] [Send].
// Method dirender sebagai pill solid berwarna semantik (gaya Bruno/Posting);
// Send sebagai tombol solid aksen. Fokus ditandai background highlight.
func (m *Model) View() string {
	if m == nil {
		return ""
	}

	boxStyle := styles.BaseBoxStyle
	if m.State == nil || m.State.TabActive == 0 {
		boxStyle = styles.ActiveBoxStyle
	}

	methodText := m.Method
	if methodText == "" {
		methodText = "GET"
	}

	// Pill method: selalu latar solid warna semantik agar method langsung
	// terbaca tanpa perlu fokus. Saat fokus, tambahkan halo chevron aksen.
	methodPill := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color(fmt.Sprint(styles.MethodColor(methodText)))).
		Bold(true).
		Padding(0, 1).
		Render(methodText)

	arrowStyle := styles.Dim()
	if m.FocusArea == 0 {
		arrowStyle = lipgloss.NewStyle().Foreground(styles.ColAccentHi).Bold(true)
	}
	leftArrow := zone.Mark("method_left_arrow", arrowStyle.Render("‹"))
	methodName := zone.Mark("method_name", methodPill)
	rightArrow := zone.Mark("method_right_arrow", arrowStyle.Render("›"))

	method := zone.Mark("method_selector", lipgloss.JoinHorizontal(lipgloss.Center, leftArrow, " ", methodName, " ", rightArrow))
	separator := styles.Dim().Render(" │ ")
	inputView := zone.Mark("url_input", m.Input.View())

	// Tombol Send: solid aksen, gelap saat non-fokus agar tidak bersaing
	// dengan pill method, cerah + underline saat fokus.
	btnStyle := lipgloss.NewStyle().
		Foreground(styles.ColBackground).
		Background(styles.ColAccent).
		Bold(true).
		Padding(0, 1)
	if m.FocusArea == 2 {
		btnStyle = btnStyle.Background(styles.ColAccentHi)
	}
	button := zone.Mark("send_button", btnStyle.Render(styles.IconSend+" Send"))

	content := lipgloss.JoinHorizontal(
		lipgloss.Center,
		method,
		separator,
		inputView,
		separator,
		button,
	)

	return zone.Mark("request_input_view", boxStyle.Render(content))
}

func RequestInputContent(m *Model) string {
	if m == nil {
		return ""
	}
	return m.View()
}

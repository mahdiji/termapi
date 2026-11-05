package ui

import (
	"fmt"
	"strings"

	"api/internal/collectionstore"
	"api/internal/ui/styles"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	zone "github.com/lrstanley/bubblezone/v2"
)

func (m AppModel) saveDialogView() string {
	screenW := max(m.ScreenWidth, 60)
	screenH := max(m.ScreenHeight, 24)
	boxW := min(max(screenW-8, 64), 100)
	contentW := boxW - 8
	rows := min(max(m.saveListRows(), 4), max(len(m.SaveVisible), 4))
	nameStyle := lipgloss.NewStyle().Foreground(styles.ColTextMuted).Bold(true)
	dim := styles.Dim()
	muted := styles.Muted()
	text := styles.Text()
	accent := styles.Title(true)

	method := "GET"
	bodyType := "none"
	authType := "none"
	url := "(URL belum diisi)"
	requestName := strings.TrimSpace(m.SaveName.Value())
	params, headers, cookies := 0, 0, 0
	if m.ReqState != nil {
		if m.ReqState.Method != "" {
			method = m.ReqState.Method
		}
		if m.ReqState.Input.Value() != "" {
			url = m.ReqState.Input.Value()
		}
		bodyType = m.ReqState.Body.Type
		if m.ReqState.Auth.Type != "" {
			authType = m.ReqState.Auth.Type
		}
		params = len(collectionstore.EnabledRows(m.ReqState.Params.Rows))
		headers = len(collectionstore.EnabledRows(m.ReqState.Headers.Rows))
		cookies = len(collectionstore.EnabledRows(m.ReqState.Cookies.Rows))
	}
	if requestName == "" {
		requestName = "New request"
	}
	if bodyType == "" {
		bodyType = "none"
	}
	if requestName == "New request" && m.ReqState != nil && strings.TrimSpace(m.ReqState.RequestName) != "" {
		requestName = strings.TrimSpace(m.ReqState.RequestName)
	}
	methodBadge := styles.MethodBadge(method).Render(" " + method + " ")

	lines := []string{
		methodBadge + "  " + accent.Render("SAVE REQUEST"),
		dim.Render("Simpan request baru ke collection atau folder yang sudah ada."),
		"",
		accent.Render("REQUEST INFO"),
		text.Render("Nama: ") + text.Render(ansi.Truncate(requestName, max(contentW-8, 1), "…")),
		text.Render("URL:  ") + muted.Render(ansi.Truncate(url, max(contentW-8, 1), "…")),
		muted.Render(fmt.Sprintf("Body %s  ·  Auth %s  ·  Params %d  ·  Headers %d  ·  Cookies %d", strings.ToUpper(bodyType), strings.ToUpper(authType), params, headers, cookies)),
		"",
		nameStyle.Render("REQUEST NAME"),
		zone.Mark("save_name", m.SaveName.View()),
		"",
		nameStyle.Render("SEARCH FOLDER") + dim.Render("  (Ctrl+F)"),
		zone.Mark("save_search", m.SaveSearch.View()),
		"",
		accent.Render(fmt.Sprintf("DESTINATION  ·  %d folder", len(m.SaveVisible))),
		styles.Dim().Render(strings.Repeat("─", contentW)),
	}

	end := min(m.SaveScroll+rows, len(m.SaveVisible))
	for row := 0; row < rows; row++ {
		index := m.SaveScroll + row
		if index >= end {
			lines = append(lines, "")
			continue
		}
		folder := m.SaveVisible[index]
		prefix := "  "
		style := text
		if index == m.SaveSelected {
			prefix = "▸ "
			style = styles.Selected().Bold(true)
		}
		label := strings.Repeat("  ", folder.Depth) + styles.IconFolderClosed + " " + folder.Name
		if folder.Depth == 0 {
			label += dim.Render("  collection")
		}
		label = ansi.Truncate(prefix+label, contentW, "…")
		if index == m.SaveSelected {
			label = style.Width(contentW).MaxWidth(contentW).Render(label)
		} else {
			label = style.Render(label)
		}
		lines = append(lines, zone.Mark(saveFolderZone(index), label))
	}

	if len(m.SaveVisible) == 0 {
		lines = append(lines, muted.Render("Tidak ada folder yang cocok."))
		lines = append(lines, dim.Render("Ketik path baru, misalnya Api/Users, lalu Enter untuk membuatnya."))
	} else if m.SaveSelected >= 0 && m.SaveSelected < len(m.SaveVisible) {
		selected := m.SaveVisible[m.SaveSelected]
		lines = append(lines, muted.Render("Lokasi: "+selected.Path))
	} else {
		lines = append(lines, "")
	}

	if m.SaveError != "" {
		lines = append(lines, lipgloss.NewStyle().Foreground(styles.ColError).Bold(true).Render("✗ "+m.SaveError))
	}
	lines = append(lines, dim.Render("Tab fokus · ↑/↓ pilih folder · Enter simpan · Esc batal"))

	content := strings.Join(lines, "\n")
	box := lipgloss.NewStyle().
		Width(contentW).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColBorderFocus).
		Background(styles.ColSurface).
		Padding(1, 3).
		Render(content)
	if lipgloss.Height(box) > screenH-2 {
		box = lipgloss.NewStyle().MaxHeight(screenH - 2).Render(box)
	}
	return lipgloss.Place(screenW, screenH, lipgloss.Center, lipgloss.Center, box)
}

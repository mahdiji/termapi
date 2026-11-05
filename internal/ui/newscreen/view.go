package newscreen

import (
	"strings"

	"api/internal/ui/styles"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// View renders the composer centered on the full screen.
func (m *Model) View() string {
	if m == nil {
		return ""
	}
	screenW := max(m.Width, 60)
	screenH := max(m.Height, 20)
	contentW := m.contentWidth()

	accent := lipgloss.NewStyle().Foreground(styles.ColAccent).Bold(true)
	accentHi := lipgloss.NewStyle().Foreground(styles.ColAccentHi).Bold(true)
	text := lipgloss.NewStyle().Foreground(styles.ColText)
	muted := lipgloss.NewStyle().Foreground(styles.ColTextMuted)
	dim := lipgloss.NewStyle().Foreground(styles.ColTextDim)
	errorSt := lipgloss.NewStyle().Foreground(styles.ColError).Bold(true)

	lines := make([]string, 0, 24)

	// Title.
	icon := styles.IconFile
	title := "NEW REQUEST"
	if m.Kind == KindFolder {
		icon = styles.IconFolderClosed
		title = "NEW FOLDER"
	}
	titleLeft := accent.Render(icon + " " + title)
	titleRight := dim.Render("Ctrl+T switch type")
	lines = append(lines, joinRow(titleLeft, titleRight, contentW))
	lines = append(lines, "")

	// Type selector.
	lines = append(lines, m.typeTabs(contentW))
	lines = append(lines, "")

	// Name.
	lines = append(lines, m.fieldLabel("Name", FocusName)+" "+m.Name.View())
	lines = append(lines, "")

	// Method (requests only).
	if m.Kind == KindRequest {
		lines = append(lines, m.methodRow())
		lines = append(lines, "")
	}

	// Search.
	searchLabel := m.fieldLabel("Find", FocusSearch)
	lines = append(lines, searchLabel+" "+m.Search.View())
	lines = append(lines, "")

	// Destination header.
	destLabel := accentHi.Render("DESTINATION")
	destValue := "→ " + m.SelectedDestinationName()
	if m.Kind == KindRequest {
		destValue += "/"
	}
	destValue = muted.Render(destValue)
	lines = append(lines, joinRow(destLabel, destValue, contentW))
	lines = append(lines, dim.Render(strings.Repeat("─", contentW)))

	// Folder list.
	rows := m.listRows()
	if len(m.visible) == 0 {
		hint := "  (no matching folders)"
		if m.Kind == KindRequest && len(m.folders) == 0 {
			hint = "  (no folders yet — Ctrl+T to create one)"
		}
		lines = append(lines, muted.Render(fit(hint, contentW)))
	} else {
		end := min(m.Scroll+rows, len(m.visible))
		for i := m.Scroll; i < end; i++ {
			dest := m.visible[i]
			marker := "  "
			if i == m.Selected {
				marker = "▸ "
			}
			prefix := marker + strings.Repeat("  ", dest.Depth)
			label := prefix + styles.IconFolderClosed + " " + dest.Name
			if dest.IsRoot {
				label += "  " + "(new top-level collection)"
			}
			label = fit(label, contentW)
			if i == m.Selected {
				style := styles.Selected().Bold(true)
				if m.Focus != FocusList {
					style = style.Background(styles.ColSurfaceAlt)
				}
				lines = append(lines, style.Width(contentW).MaxWidth(contentW).Render(label))
			} else {
				lines = append(lines, text.Render(label))
			}
		}
		for i := end - m.Scroll; i < rows; i++ {
			lines = append(lines, "")
		}
	}

	// Error.
	if m.Err != "" {
		lines = append(lines, "")
		lines = append(lines, errorSt.Render(fit("✗ "+m.Err, contentW)))
	}

	// Footer.
	lines = append(lines, "")
	footer := "Tab focus · ↑/↓ pilih · Enter buat · Ctrl+F cari · Esc batal"
	lines = append(lines, dim.Render(fit(footer, contentW)))

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.ColAccent).
		Background(styles.ColSurface).
		Padding(1, 2).
		Render(strings.Join(lines, "\n"))

	// Keep the modal inside the window even on narrow terminals.
	if lipgloss.Width(box) > screenW {
		box = ansi.Truncate(box, screenW, "")
	}
	return lipgloss.Place(screenW, screenH, lipgloss.Center, lipgloss.Center, box)
}

func (m *Model) typeTabs(width int) string {
	active := lipgloss.NewStyle().
		Foreground(styles.ColBackground).
		Background(styles.ColAccent).
		Bold(true).
		Padding(0, 1)
	inactive := lipgloss.NewStyle().
		Foreground(styles.ColTextMuted).
		Padding(0, 1)

	requestStyle, folderStyle := inactive, inactive
	if m.Kind == KindRequest {
		requestStyle = active
	} else {
		folderStyle = active
	}
	tabs := requestStyle.Render(styles.IconFile+" Request") + "  " + folderStyle.Render(styles.IconFolderClosed+" Folder")
	return joinRow(tabs, lipgloss.NewStyle().Foreground(styles.ColTextDim).Render("Enter: create"), width)
}

func (m *Model) methodRow() string {
	methodColor := lipgloss.NewStyle().Foreground(styles.MethodColor(m.Method)).Bold(true)
	label := m.fieldLabel("Method", FocusMethod)
	method := methodColor.Render(m.Method)
	chevrons := lipgloss.NewStyle().Foreground(styles.ColTextMuted).Render("‹ ")
	chevronsEnd := lipgloss.NewStyle().Foreground(styles.ColTextMuted).Render(" ›")
	pill := chevrons + method + chevronsEnd
	if m.Focus == FocusMethod {
		pill = lipgloss.NewStyle().
			Foreground(styles.ColBackground).
			Background(styles.MethodColor(m.Method)).
			Bold(true).
			Padding(0, 1).
			Render("‹ " + m.Method + " ›")
	}
	return label + " " + pill + "   " + lipgloss.NewStyle().Foreground(styles.ColTextDim).Render("←/→ change")
}

func (m *Model) fieldLabel(label string, focus Focus) string {
	style := lipgloss.NewStyle().Foreground(styles.ColTextMuted)
	marker := "  "
	if m.Focus == focus {
		style = lipgloss.NewStyle().Foreground(styles.ColAccentHi).Bold(true)
		marker = "▸ "
	}
	return style.Render(marker + padRight(label, 7))
}

func joinRow(left, right string, width int) string {
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		keep := max(width-lipgloss.Width(right)-2, 1)
		left = ansi.Truncate(left, keep, "…")
		gap = max(width-lipgloss.Width(left)-lipgloss.Width(right), 1)
	}
	return left + strings.Repeat(" ", gap) + right
}

func fit(s string, width int) string {
	if lipgloss.Width(s) > width {
		s = ansi.Truncate(s, width, "…")
	}
	if pad := width - lipgloss.Width(s); pad > 0 {
		s += strings.Repeat(" ", pad)
	}
	return s
}

func padRight(s string, width int) string {
	if pad := width - lipgloss.Width(s); pad > 0 {
		return s + strings.Repeat(" ", pad)
	}
	return s
}

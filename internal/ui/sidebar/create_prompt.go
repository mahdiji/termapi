package sidebar

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func (m *Model) CreatePromptActive() bool { return m != nil && m.CreateMode != createNone }

func (m *Model) updateCreatePrompt(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "enter":
		if m.CreateMode == createRequest && !m.CreateMethodFocus {
			m.CreateMethodFocus = true
			m.CreateInput.Blur()
			return nil
		}
		if m.CreateMode == createFolder && m.CreateHasTargetChoice && m.CreateTargetFocus {
			m.CreateTargetFocus = false
			return m.CreateInput.Focus()
		}
		m.saveCreate()
		return nil
	case "tab", "shift+tab", "up", "down":
		if m.CreateMode == createRequest {
			m.CreateMethodFocus = !m.CreateMethodFocus
			if m.CreateMethodFocus {
				m.CreateInput.Blur()
				return nil
			}
			return m.CreateInput.Focus()
		}
		if m.CreateMode == createFolder && m.CreateHasTargetChoice {
			m.CreateTargetFocus = !m.CreateTargetFocus
			if m.CreateTargetFocus {
				m.CreateInput.Blur()
				return nil
			}
			return m.CreateInput.Focus()
		}
		return nil
	case "esc", "ctrl+c":
		m.cancelCreate()
		return nil
	case "left", "h", "right", "l", "space":
		if m.CreateMode == createRequest && m.CreateMethodFocus {
			m.cycleCreateMethod(msg.String() == "right" || msg.String() == "l")
			return nil
		}
		if m.CreateMode == createFolder && m.CreateHasTargetChoice && (m.CreateTargetFocus || msg.String() == "left" || msg.String() == "right" || msg.String() == "h" || msg.String() == "l") {
			m.CreateTargetIsRoot = !m.CreateTargetIsRoot
			return nil
		}
	}
	if (m.CreateMode == createRequest && m.CreateMethodFocus) || (m.CreateMode == createFolder && m.CreateHasTargetChoice && m.CreateTargetFocus) {
		return nil
	}
	var cmd tea.Cmd
	m.CreateInput, cmd = m.CreateInput.Update(msg)
	return cmd
}
func (m *Model) renderCreatePrompt(topBorder string, border lipgloss.Style, width, height int) string {
	title := "📁 BUAT FOLDER"
	kind := "Lokasi: Di dalam collections/"
	if m.CreateMode == createRequest {
		title = "📄 BUAT REQUEST BARU"
		kind = "Lokasi: " + m.CreateParentPath
	}

	boxWidth := width - 4
	if boxWidth < 16 {
		boxWidth = 16
	}

	// Warna Tema
	accentColor := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true)
	dimColor := lipgloss.NewStyle().Foreground(lipgloss.Color("#596575"))
	highlightColor := lipgloss.NewStyle().Foreground(lipgloss.Color("#00E5FF")).Bold(true)

	lines := []string{
		accentColor.Render(title),
		dimColor.Render(kind),
		"",
	}

	// Jika buat Folder dan ada pilihan target Root vs Subfolder
	if m.CreateMode == createFolder && m.CreateHasTargetChoice {
		targetStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#AAB7C5"))
		if m.CreateTargetFocus {
			targetStyle = highlightColor.Underline(true)
		}
		targetText := "Folder Saat Ini (" + m.CreateSelectedFolderName + ")"
		if m.CreateTargetIsRoot {
			targetText = "Root Collection"
		}
		lines = append(lines, targetStyle.Render("Target:  ◀  "+targetText+"  ▶"))
		lines = append(lines, "")
	}

	// Input Nama
	lines = append(lines, m.CreateInput.View())

	// Jika buat Request, tampilkan pilihan Method
	if m.CreateMode == createRequest {
		methodStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#AAB7C5"))
		if m.CreateMethodFocus {
			methodStyle = highlightColor.Underline(true)
		}
		lines = append(lines, "")
		lines = append(lines, methodStyle.Render("Method:  ◀  "+m.CreateMethod+"  ▶"))
		lines = append(lines, "")
		lines = append(lines, dimColor.Render("Tab: Pindah   ←/→: Ubah Method"))
		lines = append(lines, dimColor.Render("Enter: Simpan   Esc: Batal"))
	} else if m.CreateMode == createFolder && m.CreateHasTargetChoice {
		lines = append(lines, "")
		lines = append(lines, dimColor.Render("Tab/↑↓/←→: Ubah Target   Enter: Simpan"))
	} else {
		lines = append(lines, "")
		lines = append(lines, dimColor.Render("Enter: Simpan   Esc: Batal"))
	}

	if m.StorageError != "" {
		lines = append(lines, "")
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("#F38BA8")).Render("✗ "+m.StorageError))
	}

	box := lipgloss.NewStyle().
		Width(boxWidth).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#FFB000")).
		Padding(1).
		Render(strings.Join(lines, "\n"))

	box = lipgloss.NewStyle().Width(width).Height(height).Align(lipgloss.Center, lipgloss.Center).Render(box)
	content := strings.Split(box, "\n")

	// Tambahkan border pinggir Sidebar
	for i := range content {
		content[i] = border.Render("│") + content[i] + border.Render("│")
	}
	for len(content) < height {
		content = append(content, border.Render("│")+strings.Repeat(" ", width)+border.Render("│"))
	}
	if len(content) > height {
		content = content[:height]
	}

	return strings.Join(append(append([]string{topBorder}, content...), border.Render("╰"+strings.Repeat("─", width)+"╯")), "\n")
}

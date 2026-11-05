package styles

import "charm.land/lipgloss/v2"

// Aturan: semua style di bawah HARUS memakai warna dari theme.go.
// Jangan tambahkan lipgloss.Color("#...") langsung di file ini.

var LogoStyle = lipgloss.NewStyle().
	Foreground(ColAccent).
	Bold(true)

var DescStyle = lipgloss.NewStyle().
	Foreground(ColTextMuted).
	MarginLeft(2)

var BaseBoxStyle = lipgloss.NewStyle().
	Border(PanelBorder).
	BorderForeground(ColBorder).
	Padding(0, PadPanel)

var ActiveBoxStyle = BaseBoxStyle.Copy().
	BorderForeground(ColBorderFocus).
	Background(ColPanelFocusBg)

var MethodStyle = lipgloss.NewStyle().
	Foreground(ColSuccess).
	Bold(true)

var ButtonStyle = lipgloss.NewStyle().
	Foreground(ColAccent).
	Bold(true)

// Title mengembalikan style judul panel. Fokus = aksen + bold.
func Title(focused bool) lipgloss.Style {
	c := ColTextMuted
	if focused {
		c = ColAccent
	}
	return lipgloss.NewStyle().Foreground(c).Bold(true)
}

func Muted() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(ColTextMuted)
}

func Dim() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(ColTextDim)
}

func Bright() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(ColTextBright)
}

func Text() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(ColText)
}

// Selected = baris terpilih: latar belakang halus + teks terang, bukan
// sekadar ganti warna teks.
func Selected() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(ColSelectionFg).
		Background(ColSelectionBg).
		Bold(true)
}

// MethodBadge merender badge method dengan warna semantik.
func MethodBadge(method string) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(MethodColor(method)).
		Bold(true)
}

// StatusBadge merender status code dengan warna semantik (2xx/3xx/4xx/5xx).
func StatusBadge(code int) lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(StatusColor(code)).
		Bold(true)
}

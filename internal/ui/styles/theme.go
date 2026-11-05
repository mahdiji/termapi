package styles

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Theme adalah satu-satunya sumber kebenaran visual TermAPI.
// Semua warna, ikon, border, dan spacing didefinisikan di sini agar
// skin (dark / light / high-contrast) bisa ditukar dari satu tempat.
//
// Prinsip desain:
//   - Satu warna aksen (cyan) untuk fokus & interaksi.
//   - Warna semantik HANYA untuk makna: status HTTP & method HTTP.
//   - Dim untuk info sekunder, bold untuk item aktif.

// ---------- Palet dasar ----------
var (
	ColBackground = lipgloss.Color("#0F1419") // base gelap, sedikit kebiruan
	ColSurface    = lipgloss.Color("#131A21") // panel surface
	ColSurfaceAlt = lipgloss.Color("#1A222C") // bar / elevated surface

	ColBorder      = lipgloss.Color("#232C38") // muted, tipis
	ColBorderFocus = lipgloss.Color("#ff0d00") // aksen saat panel fokus

	ColText       = lipgloss.Color("#C9D3DE") // teks utama
	ColTextMuted  = lipgloss.Color("#6B7A8C") // info sekunder
	ColTextDim    = lipgloss.Color("#455263") // hint tersier / disabled
	ColTextBright = lipgloss.Color("#EAF0F7") // penekanan

	// Satu aksen utama.
	ColAccent   = lipgloss.Color("#56C2D6") // cyan tenang
	ColAccentHi = lipgloss.Color("#8ADCEB") // hover / fokus kuat

	// Warna semantik untuk status HTTP & keadaan.
	ColSuccess = lipgloss.Color("#7FBF7A") // 2xx / ok
	ColInfo    = lipgloss.Color("#56C2D6") // sama dengan aksen
	ColWarning = lipgloss.Color("#D8B26E") // 3xx / peringatan
	ColClient  = lipgloss.Color("#D08A4E") // 4xx (orange)
	ColError   = lipgloss.Color("#D96A6A") // 5xx / gagal (red)

	// Seleksi: background halus, bukan hanya warna teks.
	ColSelectionBg = lipgloss.Color("#1F3A44")
	ColSelectionFg = lipgloss.Color("#EAF0F7")

	// Bar atas & bawah.
	ColBarBg    = ColSurfaceAlt
	ColStatusBg = ColSurfaceAlt

	// Panel fokus sedikit lebih terang.
	ColPanelFocusBg = lipgloss.Color("#141C25")

	// Alias legacy agar kode lama tetap kompilasi. Jangan tambah pemakaian baru.
	ColPrimary = ColWarning // logo bg
	ColAccent2 = ColAccent  // judul panel non-fokus
	ColMarker  = ColAccent  // penanda tab aktif
)

// ---------- Warna method HTTP (semantik) ----------
// GET hijau, POST kuning, PUT biru, PATCH ungu, DELETE merah.
var (
	ColMethodGET     = lipgloss.Color("#7FBF7A")
	ColMethodPOST    = lipgloss.Color("#D8B26E")
	ColMethodPUT     = lipgloss.Color("#6FA8DC")
	ColMethodPATCH   = lipgloss.Color("#B48EAD")
	ColMethodDELETE  = lipgloss.Color("#D96A6A")
	ColMethodHEAD    = lipgloss.Color("#8A97A8")
	ColMethodOPTIONS = lipgloss.Color("#8A97A8")
)

// MethodColor mengembalikan warna semantik untuk method HTTP.
func MethodColor(method string) color.Color {
	switch method {
	case "GET":
		return ColMethodGET
	case "POST":
		return ColMethodPOST
	case "PUT":
		return ColMethodPUT
	case "PATCH":
		return ColMethodPATCH
	case "DELETE":
		return ColMethodDELETE
	case "HEAD":
		return ColMethodHEAD
	case "OPTIONS":
		return ColMethodOPTIONS
	default:
		return ColTextMuted
	}
}

// StatusColor memetakan status code ke warna semantik:
// 2xx hijau, 3xx kuning, 4xx orange, 5xx merah.
func StatusColor(code int) color.Color {
	switch {
	case code >= 200 && code < 300:
		return ColSuccess
	case code >= 300 && code < 400:
		return ColWarning
	case code >= 400 && code < 500:
		return ColClient
	case code >= 500:
		return ColError
	default:
		return ColTextMuted
	}
}

// ---------- Ikon ----------
// Nerd Font glyphs dengan fallback ASCII agar tetap terbaca di terminal
// tanpa Nerd Font. Set UseNerdIcons=false untuk memakai fallback.
var UseNerdIcons = true

type iconPair struct {
	nerd string
	ascii string
}

func icon(p iconPair) string {
	if UseNerdIcons {
		return p.nerd
	}
	return p.ascii
}

var (
	IconFolderClosed = icon(iconPair{nerd: "\uf07b", ascii: "+"}) //
	IconFolderOpen   = icon(iconPair{nerd: "\uf07c", ascii: "-"})
	IconFile         = icon(iconPair{nerd: "\uf15b", ascii: "·"})
	IconChevronRight = icon(iconPair{nerd: "\uf054", ascii: "›"})
	IconChevronDown  = icon(iconPair{nerd: "\uf078", ascii: "v"})
	IconDot          = icon(iconPair{nerd: "●", ascii: "*"})
	IconCheck        = icon(iconPair{nerd: "\uf00c", ascii: "x"})
	IconSearch       = icon(iconPair{nerd: "\uf002", ascii: "/"})
	IconSend         = icon(iconPair{nerd: "\uf1d8", ascii: ">"})
	IconClock        = icon(iconPair{nerd: "\uf017", ascii: "t"})
)

// ---------- Border ----------
var PanelBorder = lipgloss.RoundedBorder()

// BorderStyle mengembalikan style border panel sesuai fokus.
func BorderStyle(focused bool) lipgloss.Style {
	c := ColBorder
	if focused {
		c = ColBorderFocus
	}
	return lipgloss.NewStyle().Foreground(c)
}

// PanelBox membungkus konten dengan border rounded; warna border mengikuti fokus.
func PanelBox(content string, width int, focused bool, title string) string {
	style := lipgloss.NewStyle().
		Border(PanelBorder).
		BorderForeground(ColBorder).
		Width(max(width, 3))
	if focused {
		style = style.BorderForeground(ColBorderFocus)
	}
	if title != "" {
		style = style.BorderTop(true)
	}
	return style.Render(content)
}

// ModalBox adalah kotak modal dengan border aksen.
var ModalBox = lipgloss.NewStyle().
	Border(PanelBorder).
	BorderForeground(ColAccent).
	Padding(1, 2)

// ---------- Spacing ----------
const (
	PadPanel = 1 // padding horizontal konsisten di dalam panel
	GapPanel = 1 // jarak antar section
)

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

package ui

import (
	"math"

	"api/internal/config"
	"api/internal/ui/nav"
	"api/internal/ui/styles"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	// minPaneHeight adalah tinggi minimum panel response.
	minPaneHeight = 6
	// minEditorHeight adalah tinggi minimum area editor request.
	minEditorHeight = 5
)

func (m AppModel) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.ScreenWidth, m.ScreenHeight = msg.Width, msg.Height
	if m.State != nil {
		m.State.Widht = msg.Width
		m.State.Height = msg.Height
	}
	if m.HelpScreen != nil {
		m.HelpScreen.SetSize(msg.Width, msg.Height)
	}
	m.recalculateLayout()
	return m, nil
}

// contentWidth mengembalikan lebar kolom kanan (request + response).
// Satu-satunya sumber kebenaran lebar agar semua tepi kanan sejajar.
func (m *AppModel) contentWidth() int {
	leftWidth := 0
	if m.Sidebar != nil {
		leftWidth = m.Sidebar.Widht
	}
	return max(m.ScreenWidth-leftWidth, 1)
}

// requestBoxHeight adalah tinggi yang dipakai request bar/editor untuk chrome.
// BaseBoxStyle punya border + padding horizontal; ambil dari frame size agar
// tidak ada offset manual yang bisa menggeser alignment.
func requestFrameHeight() int {
	return styles.BaseBoxStyle.GetVerticalFrameSize()
}

// recalculateLayout menetapkan dimensi semua pane dari satu perhitungan.
// Request/response memakai rasio (ResponseRatio) dengan clamp minimum,
// menggantikan pendekatan tinggi absolut lama.
func (m *AppModel) recalculateLayout() {
	if m == nil {
		return
	}
	navHeight := lipgloss.Height(nav.NavbarContent())
	footerHeight := 1
	mainHeight := max(m.ScreenHeight-navHeight-footerHeight, 1)
	m.MainHeight = mainHeight

	// Sidebar memakai tinggi konten yang sama dengan kolom kanan agar
	// tepi bawahnya sejajar dengan panel response di semua ukuran window.
	if m.Sidebar != nil {
		m.Sidebar.SetSize(m.ScreenWidth, mainHeight)
	}

	if m.ReqState == nil {
		return
	}

	// Lebar: satu nilai untuk semua komponen kolom kanan.
	width := m.contentWidth()
	m.ContentWidth = width
	m.ReqState.SetWidth(width)

	// ResponseRatio adalah sumber kebenaran split request/response. Tinggi
	// absolut diturunkan ulang setiap resize agar proporsi tetap terjaga.
	requestChrome := requestFrameHeight() + 3 // border editor + tab line + bar
	minRequestHeight := requestChrome + minEditorHeight
	maxRequestHeight := max(mainHeight-minPaneHeight, minRequestHeight)

	responseRatio := normalizedResponseRatio(m.ResponseRatio)
	m.ResponseRatio = responseRatio
	requestHeight := int(math.Round(float64(mainHeight) * (1 - responseRatio)))
	requestHeight = clamp(requestHeight, minRequestHeight, maxRequestHeight)

	responseHeight := mainHeight - requestHeight
	if responseHeight < minPaneHeight {
		responseHeight = minPaneHeight
		requestHeight = max(mainHeight-responseHeight, minRequestHeight)
	}
	m.RequestHeight = requestHeight
	m.ResponseHeight = responseHeight

	// Editor tumbuh mengikuti konten, min 5 baris, dibatasi requestHeight.
	editorHeight := clamp(requestHeight-requestChrome, minEditorHeight, maxRequestHeight)

	if m.Response != nil {
		m.Response.SetSize(width, responseHeight)
	}
	m.ReqState.SetEditorSize(width, editorHeight)
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// normalizedResponseRatio mengembalikan rasio response yang valid, atau
// default ketika nilai kosong/berada di luar batas aman.
func normalizedResponseRatio(ratio float64) float64 {
	if ratio <= 0.1 || ratio >= 0.9 {
		return config.DefaultResponseRatio
	}
	return ratio
}

// adjustSplit menggeser split request/response (Ctrl+Up/Down). delta positif
// membuat area request lebih tinggi. Nilai di-clamp agar response tetap
// minimal minPaneHeight dan editor tidak lebih kecil dari minEditorHeight.
func (m *AppModel) adjustSplit(delta int) {
	if m == nil {
		return
	}
	if m.MainHeight <= 0 || m.RequestHeight <= 0 {
		m.recalculateLayout()
	}
	if m.MainHeight <= 0 {
		return
	}

	requestChrome := requestFrameHeight() + 3
	minRequestHeight := requestChrome + minEditorHeight
	maxRequestHeight := max(m.MainHeight-minPaneHeight, minRequestHeight)

	requestHeight := clamp(m.RequestHeight+delta, minRequestHeight, maxRequestHeight)
	m.ResponseRatio = 1.0 - float64(requestHeight)/float64(max(m.MainHeight, 1))
	m.recalculateLayout()
	_ = config.SaveResponseRatio("", m.ResponseRatio)
}

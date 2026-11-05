package wirescreen

const (
	sparklineWidth = 20
	sparkChars     = "▁▂▃▄▅▆▇█"
)

// series adalah ring buffer angka berpanjang tetap, urutan lama -> baru.
type series struct {
	vals []float64
}

// newSeries membuat series dengan panjang tetap n, terisi 0.
func newSeries(n int) *series {
	if n < 0 {
		n = 0
	}
	return &series{vals: make([]float64, n)}
}

// push menggeser seluruh nilai ke kiri dan menaruh v di ujung kanan.
func (s *series) push(v float64) {
	if s == nil || len(s.vals) == 0 {
		return
	}
	copy(s.vals, s.vals[1:])
	s.vals[len(s.vals)-1] = v
}

// sparkline mengubah vals menjadi tepat `width` karakter dari sparkChars.
// Jika len(vals) > width ambil width nilai terakhir; jika kurang, pad kiri
// dengan "▁". Skala: min..max dari nilai yang ditampilkan; jika max==min
// semua "▁".
func sparkline(vals []float64, width int) string {
	if width <= 0 {
		return ""
	}
	bars := []rune(sparkChars)
	window := vals
	if len(window) > width {
		window = window[len(window)-width:]
	}

	out := make([]rune, 0, width)
	for i := 0; i < width-len(window); i++ {
		out = append(out, bars[0])
	}

	minV, maxV := 0.0, 0.0
	for i, v := range window {
		if i == 0 || v < minV {
			minV = v
		}
		if i == 0 || v > maxV {
			maxV = v
		}
	}
	for _, v := range window {
		if maxV == minV {
			out = append(out, bars[0])
			continue
		}
		idx := int((v - minV) / (maxV - minV) * float64(len(bars)-1))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(bars) {
			idx = len(bars) - 1
		}
		out = append(out, bars[idx])
	}
	return string(out)
}

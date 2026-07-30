package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/singoesdeep/wsl-extended/internal/ui/theme"
)

// column, tablodaki tek bir sütunu tanımlar.
type column struct {
	title string
	// width sabit genişliktir; 0 ise sütun, artan alanı diğer esnek
	// sütunlarla eşit paylaşır.
	width int
	// state true ise hücre içeriği durum rengiyle boyanır.
	state bool
}

// fitCell, hücreyi tam olarak w görünür sütuna oturtur: uzunsa kısaltır,
// kısaysa boşlukla doldurur.
func fitCell(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= w {
		return s + strings.Repeat(" ", w-lipgloss.Width(s))
	}
	if w == 1 {
		return "…"
	}
	r := []rune(s)
	for len(r) > 0 && lipgloss.Width(string(r))+1 > w {
		r = r[:len(r)-1]
	}
	out := string(r) + "…"
	return out + strings.Repeat(" ", max(0, w-lipgloss.Width(out)))
}

// resolveWidths, sabit sütunlardan artan alanı esnek sütunlara paylaştırır.
func resolveWidths(cols []column, total int) []int {
	const gap = 2 // sütunlar arası boşluk

	widths := make([]int, len(cols))
	used, flex := 0, 0
	for i, c := range cols {
		widths[i] = c.width
		if c.width == 0 {
			flex++
		} else {
			used += c.width
		}
	}
	used += gap * (len(cols) - 1)

	remaining := total - used
	if flex > 0 {
		share := remaining / flex
		if share < 8 {
			share = 8
		}
		for i, c := range cols {
			if c.width == 0 {
				widths[i] = share
			}
		}
	}
	return widths
}

// renderTable, başlık satırı + kaydırmalı gövde üretir. height, başlık dahil
// toplam satır sayısıdır.
func renderTable(cols []column, rows [][]string, cursor, width, height int) string {
	widths := resolveWidths(cols, width-2) // theme.Row'un yatay dolgusu

	var head []string
	for i, c := range cols {
		head = append(head, fitCell(c.title, widths[i]))
	}

	var b strings.Builder
	b.WriteString(theme.Header.Render(strings.Join(head, "  ")))

	visible := max(1, height-1)
	offset := 0
	if cursor >= visible {
		offset = cursor - visible + 1
	}
	end := min(len(rows), offset+visible)

	for i := offset; i < end; i++ {
		selected := i == cursor

		var cells []string
		for j, c := range cols {
			if j >= len(rows[i]) {
				cells = append(cells, fitCell("", widths[j]))
				continue
			}
			cell := fitCell(rows[i][j], widths[j])
			// Seçili satır zaten dolu arka planla çizildiği için hücre bazlı
			// renklendirme yalnızca seçili olmayan satırlara uygulanır.
			if c.state && !selected {
				cell = theme.StateStyle(strings.TrimSpace(rows[i][j])).Render(cell)
			}
			cells = append(cells, cell)
		}

		line := strings.Join(cells, "  ")
		b.WriteString("\n")
		if selected {
			b.WriteString(theme.RowSelected.Render(line))
		} else {
			b.WriteString(theme.Row.Render(line))
		}
	}

	return b.String()
}

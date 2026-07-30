package wslconf

import "strings"

// DiffLine, tek bir fark satırıdır.
type DiffLine struct {
	// Kind: ' ' değişmedi, '-' silindi, '+' eklendi.
	Kind rune
	Text string
}

// Diff, iki satır dizisi arasındaki farkı üretir.
//
// En uzun ortak alt dizi üzerinden çalışır; yapılandırma dosyaları küçük
// olduğu için O(n*m) tablo fazlasıyla yeterlidir. Amaç, kullanıcıya dosyaya
// yazmadan önce tam olarak neyin değiştiğini göstermek.
func Diff(old, new []string) []DiffLine {
	n, m := len(old), len(new)

	// lcs[i][j] = old[i:] ile new[j:] arasındaki en uzun ortak alt dizi uzunluğu
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if old[i] == new[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}

	var out []DiffLine
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case old[i] == new[j]:
			out = append(out, DiffLine{' ', old[i]})
			i, j = i+1, j+1
		case lcs[i+1][j] >= lcs[i][j+1]:
			out = append(out, DiffLine{'-', old[i]})
			i++
		default:
			out = append(out, DiffLine{'+', new[j]})
			j++
		}
	}
	for ; i < n; i++ {
		out = append(out, DiffLine{'-', old[i]})
	}
	for ; j < m; j++ {
		out = append(out, DiffLine{'+', new[j]})
	}
	return out
}

// HasChanges, farkta değişiklik olup olmadığını söyler.
func HasChanges(d []DiffLine) bool {
	for _, l := range d {
		if l.Kind != ' ' {
			return true
		}
	}
	return false
}

// FormatDiff, farkı metin olarak üretir. Değişmeyen satırlar yalnızca
// değişikliklerin çevresinde gösterilir.
func FormatDiff(d []DiffLine, context int) string {
	keep := make([]bool, len(d))
	for i, l := range d {
		if l.Kind == ' ' {
			continue
		}
		for j := max(0, i-context); j < min(len(d), i+context+1); j++ {
			keep[j] = true
		}
	}

	var b strings.Builder
	skipped := false
	for i, l := range d {
		if !keep[i] {
			skipped = true
			continue
		}
		if skipped {
			b.WriteString("  ⋮\n")
			skipped = false
		}
		b.WriteRune(l.Kind)
		b.WriteString(" ")
		b.WriteString(l.Text)
		b.WriteString("\n")
	}
	return strings.TrimSuffix(b.String(), "\n")
}

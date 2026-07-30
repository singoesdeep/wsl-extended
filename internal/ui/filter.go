package ui

import "strings"

// filterText, satırın filtreyle eşleşip eşleşmediğini söyler. Eşleşme büyük/
// küçük harf duyarsızdır ve parçaların herhangi birinde geçmesi yeterlidir.
func filterText(query string, parts ...string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return true
	}
	for _, p := range parts {
		if strings.Contains(strings.ToLower(p), q) {
			return true
		}
	}
	return false
}

// visible, etkin filtreye göre görünen satırların gerçek indekslerini verir.
//
// İmleç ve işaretleme her zaman gerçek indeksler üzerinden çalışır; filtre
// yalnızca hangi satırların çizileceğini belirler. Böylece filtre değişince
// seçili öğe kaybolmaz ve işlem yanlış hedefe uygulanmaz.
func (m Model) visible(t tabID) []int {
	q := m.filter

	var out []int
	switch t {
	case tabDistros:
		for i, d := range m.distros {
			if filterText(q, d.Name, m.displayName(d.Name), string(d.State)) {
				out = append(out, i)
			}
		}
	case tabStore:
		for i, o := range m.online {
			if filterText(q, o.Name, o.Friendly) {
				out = append(out, i)
			}
		}
	case tabContainers:
		for i, c := range m.containers {
			if filterText(q, c.Name(), c.Image.String(), c.State.String()) {
				out = append(out, i)
			}
		}
	case tabImages:
		for i, im := range m.images {
			if filterText(q, im.Repository.String(), im.Tag.String(), im.ID.String()) {
				out = append(out, i)
			}
		}
	case tabVolumes:
		for i, v := range m.volumes {
			if filterText(q, v.Name.String(), v.Driver.String()) {
				out = append(out, i)
			}
		}
	case tabNetworks:
		for i, n := range m.networks {
			if filterText(q, n.Name.String(), n.Driver.String(), n.Scope.String()) {
				out = append(out, i)
			}
		}
	}
	return out
}

// realIndex, etkin sekmede imlecin gösterdiği gerçek satır indeksini verir.
func (m Model) realIndex(t tabID) (int, bool) {
	vis := m.visible(t)
	c := m.cursors[t]
	if c < 0 || c >= len(vis) {
		return 0, false
	}
	return vis[c], true
}

// markedOrCurrent, işlemin uygulanacağı gerçek indeksleri verir: işaretli
// satırlar varsa onlar, yoksa yalnızca imlecin altındaki satır.
func (m Model) markedOrCurrent(t tabID) []int {
	if marks := m.marked[t]; len(marks) > 0 {
		// Görünen sırayla döndürülür ki onay ekranındaki liste ekrandakiyle
		// aynı sırada olsun.
		var out []int
		for _, i := range m.visible(t) {
			if marks[i] {
				out = append(out, i)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	if i, ok := m.realIndex(t); ok {
		return []int{i}
	}
	return nil
}

// toggleMark, imlecin altındaki satırın işaretini değiştirir.
func (m *Model) toggleMark() {
	i, ok := m.realIndex(m.active)
	if !ok {
		return
	}
	if m.marked[m.active] == nil {
		m.marked[m.active] = map[int]bool{}
	}
	if m.marked[m.active][i] {
		delete(m.marked[m.active], i)
		return
	}
	m.marked[m.active][i] = true
}

// clearMarks, sekmenin işaretlerini temizler. Liste yenilendiğinde indeksler
// kayabileceği için işlem sonrası çağrılır.
func (m *Model) clearMarks(t tabID) {
	delete(m.marked, t)
}

func (m Model) markCount(t tabID) int { return len(m.marked[t]) }

func (m Model) isMarked(t tabID, real int) bool { return m.marked[t][real] }

package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/singoesdeep/wsl-extended/internal/wslc"
)

func logsWith(lines ...string) logModel {
	m := logModel{active: true, target: "web", follow: true}
	for _, l := range lines {
		m = m.append(l)
	}
	return m
}

// Takip açıkken panel her zaman son satırları göstermeli.
func TestLogsFollowShowsTail(t *testing.T) {
	m := logsWith("1", "2", "3", "4", "5")

	got := m.visibleLines(3)
	want := []string{"3", "4", "5"}

	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("görünen satırlar = %q, %q bekleniyordu", got, want)
	}
}

// Elle kaydırmak takibi bırakmalı; yoksa okurken ekran sürekli altına kaçar.
func TestLogsManualScrollStopsFollowing(t *testing.T) {
	m := logsWith("1", "2", "3", "4", "5")

	m, closed := m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}, 3)
	if closed {
		t.Fatal("k tuşu paneli kapattı")
	}
	if m.follow {
		t.Error("elle kaydırma sonrası takip hâlâ açık")
	}

	// G takibe geri döndürür.
	m, _ = m.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}, 3)
	if !m.follow {
		t.Error("G takibi geri açmadı")
	}
}

// Panel uzun süre açık kalınca bellek sınırsız büyümemeli.
func TestLogsRingBufferCaps(t *testing.T) {
	m := logModel{active: true, follow: true}
	for i := 0; i < logCapacity+500; i++ {
		m = m.append("satır")
	}

	if len(m.lines) != logCapacity {
		t.Errorf("satır sayısı = %d, en fazla %d olmalıydı", len(m.lines), logCapacity)
	}
}

func TestLogsCloseKeys(t *testing.T) {
	for _, key := range []string{"esc", "q", "L"} {
		m := logsWith("a")

		var msg tea.KeyMsg
		if key == "esc" {
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		} else {
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
		}

		if _, closed := m.update(msg, 10); !closed {
			t.Errorf("%q paneli kapatmadı", key)
		}
	}
}

func TestLogsViewShowsTargetAndLines(t *testing.T) {
	out := logsWith("merhaba dünya").view(80, 10)

	if !strings.Contains(out, "web") {
		t.Errorf("başlıkta hedef adı yok:\n%s", out)
	}
	if !strings.Contains(out, "merhaba dünya") {
		t.Errorf("günlük satırı görünmüyor:\n%s", out)
	}
}

func TestLogsEmptyState(t *testing.T) {
	m := logModel{active: true, target: "web", follow: true}
	if out := m.view(80, 10); !strings.Contains(out, "Henüz günlük satırı yok") {
		t.Errorf("boş durum mesajı bekleniyordu:\n%s", out)
	}
}

// Panel açıkken tuşlar arkadaki listeye sızmamalı.
func TestLogPanelSwallowsKeys(t *testing.T) {
	m := testModel()
	m.active = tabContainers
	m.containers = []wslc.Container{{Names: "web"}, {Names: "api"}}
	m.logs = logsWith("a", "b")

	before := m.cursors[tabContainers]
	updated, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)

	if m.cursors[tabContainers] != before {
		t.Error("günlük paneli açıkken imleç hareket etti")
	}
}

// Çalışmayan bir kapsayıcıda kabuk açmaya çalışmak, komutu çalıştırmak yerine
// kullanıcıyı uyarmalı.
func TestShellOnStoppedContainerWarns(t *testing.T) {
	m := testModel()
	m.active = tabContainers
	m.containers = []wslc.Container{{Names: "web", State: "exited"}}

	updated, cmd := m.openShell()
	m = updated.(Model)

	if cmd != nil {
		t.Error("durmuş kapsayıcı için kabuk komutu üretildi")
	}
	if !m.noticeErr || !strings.Contains(m.notice, "çalışmıyor") {
		t.Errorf("uyarı bekleniyordu, bildirim = %q", m.notice)
	}
}

func TestStatsPanelClosesOnEsc(t *testing.T) {
	m := testModel()
	m.stats = statsModel{active: true}

	updated, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	if m.stats.active {
		t.Error("esc istatistik panelini kapatmadı")
	}
}

func TestStatsViewRendersRows(t *testing.T) {
	m := statsModel{active: true, items: []wslc.Stat{
		{Name: "web", CPUPerc: "12.5%", MemUsage: "48MiB / 2GiB", PIDs: "7"},
	}}

	out := m.view(120, 10)
	for _, want := range []string{"web", "12.5%", "48MiB"} {
		if !strings.Contains(out, want) {
			t.Errorf("istatistik çıktısında %q yok:\n%s", want, out)
		}
	}
}

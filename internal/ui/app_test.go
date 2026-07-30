package ui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/singoesdeep/wsl-extended/internal/store"
	"github.com/singoesdeep/wsl-extended/internal/wsl"
	"github.com/singoesdeep/wsl-extended/internal/wslc"
)

var errTest = errors.New("komut patladı")

func testModel() Model {
	m := New()
	m.wslOK, m.wslcOK = true, true
	m.width, m.height = 120, 30
	// Testler kullanıcının gerçek veri dosyasına yazmamalı.
	m.data = &store.Data{Aliases: map[string]string{}}
	m.dataPath = filepath.Join(os.TempDir(), "wsl-extended-test", "data.json")
	return m
}

func TestViewRendersDistroRow(t *testing.T) {
	m := testModel()
	m.distros = []wsl.Distro{
		{Name: "FedoraLinux-44", State: wsl.StateStopped, Version: "2", Default: true},
	}

	out := m.View()
	for _, want := range []string{"FedoraLinux-44", "Stopped", "Distros"} {
		if !strings.Contains(out, want) {
			t.Errorf("çıktıda %q yok:\n%s", want, out)
		}
	}
}

func TestViewShowsEmptyState(t *testing.T) {
	m := testModel()
	if out := m.View(); !strings.Contains(out, "Kurulu distro yok") {
		t.Errorf("boş durum mesajı bekleniyordu:\n%s", out)
	}
}

// wslc kurulu değilken kapsayıcı sekmeleri çökmemeli, bilgi göstermeli.
func TestViewWithoutWSLC(t *testing.T) {
	m := testModel()
	m.wslcOK = false
	m.active = tabContainers

	if out := m.View(); !strings.Contains(out, "wslc.exe bulunamadı") {
		t.Errorf("wslc yok bilgisi bekleniyordu:\n%s", out)
	}
}

func TestViewEveryTabRenders(t *testing.T) {
	m := testModel()
	m.distros = []wsl.Distro{{Name: "d", State: wsl.StateRunning, Version: "2"}}
	m.containers = []wslc.Container{{ID: "abc123", Names: "web", Image: "nginx", State: "running"}}
	m.images = []wslc.Image{{Repository: "nginx", Tag: "latest", ID: "sha256:aa"}}
	m.volumes = []wslc.Volume{{Name: "data", Driver: "local"}}
	m.networks = []wslc.Network{{Name: "bridge", Driver: "bridge"}}

	for tab := tabDistros; tab < tabCount; tab++ {
		m.active = tab
		if out := m.View(); strings.TrimSpace(out) == "" {
			t.Errorf("%s sekmesi boş çizdi", tabNames[tab])
		}
	}
}

func TestCursorStaysInBounds(t *testing.T) {
	m := testModel()
	m.distros = []wsl.Distro{{Name: "a"}, {Name: "b"}}

	// Liste sonundan taşma denemesi.
	for i := 0; i < 5; i++ {
		updated, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
		m = updated.(Model)
	}
	if m.cursors[tabDistros] != 1 {
		t.Errorf("imleç = %d; son satırda kalmalıydı", m.cursors[tabDistros])
	}

	// Liste kısaldığında imleç geri çekilmeli.
	m.distros = m.distros[:1]
	m.clampCursor(tabDistros)
	if m.cursors[tabDistros] != 0 {
		t.Errorf("imleç = %d; liste kısalınca 0 olmalıydı", m.cursors[tabDistros])
	}
}

func TestTabNavigationWraps(t *testing.T) {
	m := testModel()
	updated, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = updated.(Model)

	if m.active != tabNetworks {
		t.Errorf("aktif sekme = %v; ilk sekmeden geriye gidince sona sarmalıydı", m.active)
	}
}

// Onay diyaloğu açıkken tuşlar arkadaki listeye sızmamalı; aksi hâlde onay
// beklerken imleç kayar ve işlem yanlış hedefe uygulanır.
func TestKeysDoNotLeakThroughConfirm(t *testing.T) {
	m := testModel()
	m.distros = []wsl.Distro{{Name: "alpha"}, {Name: "beta"}}

	updated, _ := m.handleKey(runes("d"))
	m = updated.(Model)
	if !m.confirm.active {
		t.Fatal("d tuşu onay diyaloğunu açmadı")
	}

	before := m.cursors[tabDistros]
	updated, _ = m.handleKey(runes("j"))
	m = updated.(Model)

	if m.cursors[tabDistros] != before {
		t.Error("onay açıkken imleç hareket etti")
	}
	if !m.confirm.active {
		t.Error("j tuşu diyalogu kapattı")
	}
}

// Bir işlem sürerken yeni bir işlem başlatılamamalı.
func TestNoNewActionWhileBusy(t *testing.T) {
	m := testModel()
	m.distros = []wsl.Distro{{Name: "alpha"}}
	m.busy = true

	updated, _ := m.handleKey(runes("d"))
	m = updated.(Model)

	if m.confirm.active {
		t.Error("işlem sürerken yeni onay diyaloğu açıldı")
	}
}

func TestActionFailureShowsError(t *testing.T) {
	m := testModel()
	m.busy = true

	updated, _ := m.Update(actionDoneMsg{
		act: action{done: "silindi"},
		err: errTest,
	})
	m = updated.(Model)

	if m.busy {
		t.Error("işlem bitti ama busy true kaldı")
	}
	if !m.noticeErr || !strings.Contains(m.notice, "patladı") {
		t.Errorf("hata bildirimi bekleniyordu: %q", m.notice)
	}
}

func TestActionSuccessShowsNotice(t *testing.T) {
	m := testModel()
	m.busy = true

	updated, _ := m.Update(actionDoneMsg{act: action{done: "alpha silindi"}})
	m = updated.(Model)

	if m.noticeErr {
		t.Error("başarılı işlem hata olarak işaretlendi")
	}
	if m.notice != "alpha silindi" {
		t.Errorf("bildirim = %q", m.notice)
	}
}

func TestFitCellTruncatesAndPads(t *testing.T) {
	if got := fitCell("kısa", 10); len([]rune(got)) != 10 {
		t.Errorf("fitCell dolgu yapmadı: %q", got)
	}
	got := fitCell("çok uzun bir hücre içeriği", 10)
	if !strings.HasSuffix(strings.TrimRight(got, " "), "…") {
		t.Errorf("fitCell kısaltma işareti koymadı: %q", got)
	}
}

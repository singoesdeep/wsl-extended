package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/singoesdeep/wsl-extended/internal/wsl"
	"github.com/singoesdeep/wsl-extended/internal/wslc"
)

func threeDistros() Model {
	m := testModel()
	m.distros = []wsl.Distro{
		{Name: "Ubuntu-24.04", State: wsl.StateStopped},
		{Name: "FedoraLinux-44", State: wsl.StateRunning},
		{Name: "Debian", State: wsl.StateStopped},
	}
	return m
}

func TestFilterNarrowsList(t *testing.T) {
	m := threeDistros()
	m.filter = "fedora" // büyük/küçük harf duyarsız

	if got := m.count(tabDistros); got != 1 {
		t.Fatalf("görünen satır = %d, 1 bekleniyordu", got)
	}

	_, rows, _ := m.tableData()
	if len(rows) != 1 || !strings.Contains(rows[0][0], "FedoraLinux-44") {
		t.Errorf("yanlış satır gösterildi: %+v", rows)
	}
}

// Filtre uygulanmışken imleç görünen satırlar arasında gezinir ama işlem gerçek
// kayda uygulanmalı. Bu karışırsa yanlış distro silinir.
func TestFilteredActionTargetsCorrectRow(t *testing.T) {
	m := threeDistros()
	m.filter = "debian"
	m.cursors[tabDistros] = 0

	d, ok := m.selectedDistro()
	if !ok {
		t.Fatal("seçili distro yok")
	}
	if d.Name != "Debian" {
		t.Fatalf("seçili distro = %q; süzgeçteki satır olmalıydı", d.Name)
	}

	act, ok := m.actionFor("d")
	if !ok {
		t.Fatal("işlem üretilmedi")
	}
	if act.target != "Debian" || act.confirmWord != "Debian" {
		t.Errorf("işlem yanlış hedefe kuruldu: %+v", act)
	}
}

// Filtre takma adla da eşleşmeli: kullanıcı gördüğü adı arar.
func TestFilterMatchesAlias(t *testing.T) {
	m := threeDistros()
	m.data.SetAlias("FedoraLinux-44", "iş makinesi")
	m.filter = "makine"

	if got := m.count(tabDistros); got != 1 {
		t.Fatalf("takma adla eşleşme başarısız, görünen = %d", got)
	}
	if d, _ := m.selectedDistro(); d.Name != "FedoraLinux-44" {
		t.Errorf("seçili = %+v", d)
	}
}

func TestFilterTypingGoesToQueryNotShortcuts(t *testing.T) {
	m := threeDistros()

	updated, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = updated.(Model)
	if !m.filtering {
		t.Fatal("/ filtre kipini açmadı")
	}

	// "d" normalde silme onayı açar; filtre kipinde metne yazılmalı.
	for _, r := range "deb" {
		updated, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = updated.(Model)
	}

	if m.confirm.active {
		t.Error("filtre yazarken silme onayı açıldı")
	}
	if m.filter != "deb" {
		t.Errorf("filtre = %q", m.filter)
	}

	// enter gezinmeye döner ama filtreyi korur.
	updated, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	if m.filtering || m.filter != "deb" {
		t.Errorf("enter sonrası filtering=%v filter=%q", m.filtering, m.filter)
	}
}

func TestFilterEscapeClears(t *testing.T) {
	m := threeDistros()
	m.filtering, m.filter = true, "deb"

	updated, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	if m.filter != "" || m.filtering {
		t.Errorf("esc filtreyi temizlemedi: %q %v", m.filter, m.filtering)
	}
}

// Sekme değişince filtre sıfırlanmalı; aksi hâlde diğer sekme boş görünür.
func TestFilterResetsOnTabChange(t *testing.T) {
	m := threeDistros()
	m.filter = "fedora"

	updated, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)

	if m.filter != "" {
		t.Errorf("sekme değişince filtre kaldı: %q", m.filter)
	}
}

func TestMarkAndBulkAction(t *testing.T) {
	m := testModel()
	m.active = tabContainers
	m.containers = []wslc.Container{
		{Names: "web", State: "running"},
		{Names: "api", State: "running"},
		{Names: "db", State: "running"},
	}

	// İlk iki satırı işaretle.
	m.cursors[tabContainers] = 0
	m.toggleMark()
	m.cursors[tabContainers] = 1
	m.toggleMark()

	if m.markCount(tabContainers) != 2 {
		t.Fatalf("işaretli sayısı = %d", m.markCount(tabContainers))
	}

	act, ok := m.actionFor("s")
	if !ok {
		t.Fatal("işlem üretilmedi")
	}
	act = m.applyBulk(act)

	if len(act.targets) != 2 {
		t.Fatalf("hedef sayısı = %d: %+v", len(act.targets), act.targets)
	}
	if act.targets[0] != "web" || act.targets[1] != "api" {
		t.Errorf("hedefler = %+v", act.targets)
	}
	if !strings.Contains(act.body, "web") || !strings.Contains(act.body, "api") {
		t.Errorf("onay metni hedefleri listelemiyor:\n%s", act.body)
	}
}

// Geri dönüşü olmayan işlemler toplu yapılmamalı: ad yazdırma koruması tek
// hedefe göre kuruludur.
func TestBulkSkipsIrreversibleActions(t *testing.T) {
	m := threeDistros()
	m.cursors[tabDistros] = 0
	m.toggleMark()
	m.cursors[tabDistros] = 1
	m.toggleMark()

	act, ok := m.actionFor("d") // distro unregister
	if !ok {
		t.Fatal("işlem üretilmedi")
	}
	act = m.applyBulk(act)

	if len(act.targets) != 0 {
		t.Errorf("distro silme toplu hâle getirildi: %+v", act.targets)
	}
	if act.confirmWord == "" {
		t.Error("ad yazdırma koruması kayboldu")
	}
}

func TestMarkToggleOff(t *testing.T) {
	m := threeDistros()
	m.toggleMark()
	if m.markCount(tabDistros) != 1 {
		t.Fatal("işaret konmadı")
	}
	m.toggleMark()
	if m.markCount(tabDistros) != 0 {
		t.Error("ikinci basış işareti kaldırmadı")
	}
}

// Yardım satırı terminale sığmalı; taşarsa satır kırılıp düzen bozulur.
func TestHelpLineFitsWidth(t *testing.T) {
	for _, width := range []int{60, 80, 100, 140} {
		m := threeDistros()
		m.width = width

		got := lipgloss.Width(m.viewHelp())
		if got > width {
			t.Errorf("genişlik %d için yardım satırı %d sütun: taşıyor\n%s",
				width, got, m.viewHelp())
		}
	}
}

// Dar ekranda kırpma olduğunda kullanıcı tam listeye yönlendirilmeli.
func TestHelpLineOffersFullList(t *testing.T) {
	m := threeDistros()
	m.width = 60

	if !strings.Contains(m.viewHelp(), "tümü") {
		t.Errorf("kırpılmış yardım satırı tam listeye yönlendirmiyor:\n%s", m.viewHelp())
	}
}

// Tuş kısayolları küçük harf olmalı; büyük harfler zor erişilir.
func TestShortcutsAreLowercase(t *testing.T) {
	for _, tab := range []tabID{tabDistros, tabStore, tabContainers, tabImages} {
		m := threeDistros()
		m.active = tab

		for _, k := range m.contextKeys() {
			key := k[0]
			if key != strings.ToLower(key) {
				t.Errorf("%s sekmesinde büyük harfli kısayol: %q", tabNames[tab], key)
			}
		}
	}
}

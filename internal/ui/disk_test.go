package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/singoesdeep/wsl-extended/internal/wsl"
)

func storeModel() Model {
	m := testModel()
	m.active = tabStore
	m.online = []wsl.OnlineDistro{
		{Name: "Ubuntu-24.04", Friendly: "Ubuntu 24.04 LTS"},
		{Name: "FedoraLinux-44", Friendly: "Fedora Linux 44"},
	}
	m.distros = []wsl.Distro{{Name: "FedoraLinux-44", State: wsl.StateStopped}}
	return m
}

// Kurulu dağıtımlar katalogda işaretlenmeli.
func TestStoreMarksInstalled(t *testing.T) {
	m := storeModel()

	_, rows, _ := m.tableData()
	if len(rows) != 2 {
		t.Fatalf("2 satır bekleniyordu, %d bulundu", len(rows))
	}
	if rows[0][2] != "" {
		t.Errorf("kurulu olmayan işaretlendi: %q", rows[0][2])
	}
	if rows[1][2] != "kurulu" {
		t.Errorf("kurulu dağıtım işaretlenmedi: %q", rows[1][2])
	}
}

func TestStoreInstallOpensConfirm(t *testing.T) {
	m := storeModel()
	m.cursors[tabStore] = 0

	updated, _ := m.confirmInstall()
	m = updated.(Model)

	if !m.confirm.active {
		t.Fatal("onay diyaloğu açılmadı")
	}
	if m.confirm.act.kind != actDistroInstall {
		t.Errorf("kind = %v", m.confirm.act.kind)
	}
	if m.confirm.act.target != "Ubuntu-24.04" {
		t.Errorf("hedef = %q", m.confirm.act.target)
	}
	// Kurulumdan sonra dağıtımın açılmayacağı kullanıcıya söylenmeli.
	if !strings.Contains(m.confirm.act.body, "otomatik açılmaz") {
		t.Errorf("onay metni --no-launch davranışını anlatmıyor:\n%s", m.confirm.act.body)
	}
}

// Zaten kurulu bir dağıtım için kurulum başlatılmamalı.
func TestStoreRejectsInstalledDistro(t *testing.T) {
	m := storeModel()
	m.cursors[tabStore] = 1

	updated, _ := m.confirmInstall()
	m = updated.(Model)

	if m.confirm.active {
		t.Error("kurulu dağıtım için onay açıldı")
	}
	if !m.noticeErr || !strings.Contains(m.notice, "zaten kurulu") {
		t.Errorf("uyarı bekleniyordu: %q", m.notice)
	}
}

// Mağaza wslc gerektirmemeli; wslc yokken de çalışmalı.
func TestStoreDoesNotNeedWSLC(t *testing.T) {
	if tabStore.needsWSLC() {
		t.Error("mağaza sekmesi wslc gerektiriyor olarak işaretlenmiş")
	}
	if tabDistros.needsWSLC() {
		t.Error("distro sekmesi wslc gerektiriyor olarak işaretlenmiş")
	}
	if !tabContainers.needsWSLC() {
		t.Error("kapsayıcı sekmesi wslc gerektirmiyor olarak işaretlenmiş")
	}
}

func diskModel(t *testing.T, running bool) Model {
	t.Helper()

	state := wsl.StateStopped
	if running {
		state = wsl.StateRunning
	}

	m := testModel()
	m.distros = []wsl.Distro{{Name: "alpha", State: state}}
	m.menu = diskMenu("alpha", running)
	return m
}

func TestDiskMenuResizeOpensPrompt(t *testing.T) {
	m := diskModel(t, false)

	updated, _ := m.handleDiskChoice("resize")
	m = updated.(Model)

	if !m.prompt.active || m.prompt.kind != promptResize {
		t.Fatal("boyut formu açılmadı")
	}
	if m.prompt.subject != "alpha" {
		t.Errorf("hedef = %q", m.prompt.subject)
	}
}

func TestDiskResizeValidatesSize(t *testing.T) {
	m := diskModel(t, false)
	updated, _ := m.handleDiskChoice("resize")
	m = updated.(Model)

	m.prompt = setField(m.prompt, 0, "çok büyük")
	m, _ = m.submitPrompt()

	if m.confirm.active {
		t.Error("geçersiz boyutla onay açıldı")
	}
	if m.prompt.err == "" {
		t.Error("hata gösterilmedi")
	}

	// Geçerli değer kabul edilmeli.
	m.prompt = setField(m.prompt, 0, "60GB")
	m, _ = m.submitPrompt()

	if !m.confirm.active {
		t.Fatalf("geçerli boyut reddedildi: %q", m.prompt.err)
	}
	if m.confirm.act.size != "60GB" {
		t.Errorf("boyut = %q", m.confirm.act.size)
	}
}

func TestDiskSparseTogglesProduceActions(t *testing.T) {
	m := diskModel(t, false)

	updated, _ := m.handleDiskChoice("sparse-on")
	m = updated.(Model)
	if !m.confirm.active || m.confirm.act.kind != actDistroSparse || !m.confirm.act.sparse {
		t.Errorf("seyrek disk açma işi yanlış: %+v", m.confirm.act)
	}

	m = diskModel(t, false)
	updated, _ = m.handleDiskChoice("sparse-off")
	m = updated.(Model)
	if !m.confirm.active || m.confirm.act.sparse {
		t.Errorf("seyrek disk kapatma işi yanlış: %+v", m.confirm.act)
	}
}

func TestDiskMoveProducesAction(t *testing.T) {
	m := diskModel(t, false)
	updated, _ := m.handleDiskChoice("move")
	m = updated.(Model)

	m.prompt = setField(m.prompt, 0, `D:\wsl\alpha`)
	m, _ = m.submitPrompt()

	if !m.confirm.active {
		t.Fatalf("onay açılmadı: %q", m.prompt.err)
	}
	if m.confirm.act.kind != actDistroMove || m.confirm.act.path != `D:\wsl\alpha` {
		t.Errorf("taşıma işi yanlış: %+v", m.confirm.act)
	}
}

// resize ve move distro kapalıyken çalışır; menü bunu önceden söylemeli.
func TestDiskMenuWarnsWhenRunning(t *testing.T) {
	out := diskMenu("alpha", true).view(100)
	if !strings.Contains(out, "önce s ile durdur") {
		t.Errorf("çalışan distro uyarısı yok:\n%s", out)
	}

	out = diskMenu("alpha", false).view(100)
	if strings.Contains(out, "önce s ile durdur") {
		t.Errorf("durmuş distroda gereksiz uyarı var:\n%s", out)
	}
}

func TestDiskMenuSwallowsKeys(t *testing.T) {
	m := diskModel(t, false)

	before := m.cursors[tabDistros]
	updated, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)

	if m.cursors[tabDistros] != before {
		t.Error("menü açıkken imleç listede hareket etti")
	}
	if !m.menu.active {
		t.Error("j tuşu menüyü kapattı")
	}
	if m.menu.idx != 1 {
		t.Errorf("menü imleci = %d", m.menu.idx)
	}
}

// Başlat/durdur kararı, tazelenen duruma göre verilmeli. Liste bayatken
// çalışan bir distro için "başlat" sormak kullanıcıyı yanıltır.
func TestToggleUsesRefreshedState(t *testing.T) {
	m := testModel()
	// Ekrandaki (bayat) durum: durmuş.
	m.distros = []wsl.Distro{{Name: "alpha", State: wsl.StateStopped}}

	// Tazeleme distronun aslında çalıştığını söylüyor.
	updated, _ := m.Update(actionAfterRefreshMsg{
		key:     "s",
		distros: []wsl.Distro{{Name: "alpha", State: wsl.StateRunning}},
	})
	m = updated.(Model)

	if !m.confirm.active {
		t.Fatal("onay diyaloğu açılmadı")
	}
	if m.confirm.act.kind != actDistroStop {
		t.Errorf("kind = %v; çalışan distro için durdurma beklenirdi", m.confirm.act.kind)
	}
	if !strings.Contains(m.confirm.act.done, "durduruldu") {
		t.Errorf("bildirim = %q", m.confirm.act.done)
	}
}

func TestToggleRefreshFailureFallsBackToKnownState(t *testing.T) {
	m := testModel()
	m.distros = []wsl.Distro{{Name: "alpha", State: wsl.StateRunning}}

	// Tazeleme başarısız: eldeki durum korunmalı, işlem yine de kurulmalı.
	updated, _ := m.Update(actionAfterRefreshMsg{key: "s", err: errTest})
	m = updated.(Model)

	if !m.confirm.active {
		t.Fatal("tazeleme hatasında işlem hiç kurulmadı")
	}
	if m.confirm.act.kind != actDistroStop {
		t.Errorf("kind = %v", m.confirm.act.kind)
	}
}

// Distro çalışıyor ama canlı bilgi alınamadıysa panel "çalışmıyor" dememeli.
func TestInspectRunningButNoLiveData(t *testing.T) {
	m := inspectModel{active: true, info: wsl.Info{
		Name: "alpha", State: wsl.StateRunning, Live: false,
		LiveErr: errTest,
	}}

	out := m.view(100, 24)
	if strings.Contains(out, "Distro çalışmıyor") {
		t.Errorf("çalışan distro için 'çalışmıyor' yazdı:\n%s", out)
	}
	if !strings.Contains(out, "canlı bilgi alınamadı") {
		t.Errorf("gerçek neden gösterilmiyor:\n%s", out)
	}
}

// Başlatma onayı, WSL'in boştaki distroyu kapatacağını söylemeli.
func TestStartConfirmExplainsIdleShutdown(t *testing.T) {
	m := testModel()
	m.distros = []wsl.Distro{{Name: "alpha", State: wsl.StateStopped}}

	act, ok := m.actionFor("s")
	if !ok {
		t.Fatal("işlem üretilmedi")
	}
	if !strings.Contains(act.body, "kendiliğinden kapatır") {
		t.Errorf("boşta kapanma uyarısı yok:\n%s", act.body)
	}
}

// Detay paneli, distro çalışmıyorken canlı bilgi vaat etmemeli.
func TestInspectViewStoppedDistro(t *testing.T) {
	m := inspectModel{active: true, info: wsl.Info{
		Name: "alpha", State: wsl.StateStopped, Version: "2",
		BasePath: `C:\wsl\alpha`, DiskPath: `C:\wsl\alpha\ext4.vhdx`,
		DiskSize: 5 * 1024 * 1024 * 1024,
	}}

	out := m.view(100, 24)
	if !strings.Contains(out, "5.0 GB") {
		t.Errorf("disk boyutu görünmüyor:\n%s", out)
	}
	if !strings.Contains(out, "s ile başlat") {
		t.Errorf("canlı bilgi için başlatma ipucu yok:\n%s", out)
	}
}

func TestInspectViewRunningDistro(t *testing.T) {
	m := inspectModel{active: true, alias: "iş", info: wsl.Info{
		Name: "alpha", State: wsl.StateRunning, Live: true,
		Kernel: "6.18.35.2-1", IP: "172.20.1.5",
		DiskUsed: "3.1G", DiskFree: "20G", DiskUse: "14%",
	}}

	out := m.view(100, 24)
	for _, want := range []string{"6.18.35.2-1", "172.20.1.5", "3.1G", "iş"} {
		if !strings.Contains(out, want) {
			t.Errorf("çıktıda %q yok:\n%s", want, out)
		}
	}
}

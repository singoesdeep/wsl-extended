package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/singoesdeep/wsl-extended/internal/wsl"
)

// exportModel, distro sekmesinde export formu açılmış bir model kurar.
func exportModel(t *testing.T, d wsl.Distro) Model {
	t.Helper()

	m := testModel()
	m.distros = []wsl.Distro{d}
	m.prompt = newExportPrompt(d)
	return m
}

func setField(p promptModel, idx int, value string) promptModel {
	p.idx = idx
	p.fields[idx].value = value
	return p
}

// Çalışan bir distro export edilirken önce durdurulmalı; yoksa dosya sistemi
// arşivlenirken değişir ve yedek tutarsız olur.
func TestExportStopsRunningDistroFirst(t *testing.T) {
	dir := t.TempDir()
	m := exportModel(t, wsl.Distro{Name: "alpha", State: wsl.StateRunning})
	m.prompt = setField(m.prompt, 0, filepath.Join(dir, "alpha.tar"))

	m, _ = m.submitPrompt()

	if !m.confirm.active {
		t.Fatal("onay diyaloğu açılmadı")
	}
	if !m.confirm.act.stopFirst {
		t.Error("çalışan distro için stopFirst false")
	}
	if !strings.Contains(m.confirm.act.body, "durdurulacak") {
		t.Errorf("onay metni durdurmadan söz etmiyor:\n%s", m.confirm.act.body)
	}
}

func TestExportDoesNotStopStoppedDistro(t *testing.T) {
	dir := t.TempDir()
	m := exportModel(t, wsl.Distro{Name: "alpha", State: wsl.StateStopped})
	m.prompt = setField(m.prompt, 0, filepath.Join(dir, "alpha.tar"))

	m, _ = m.submitPrompt()

	if m.confirm.act.stopFirst {
		t.Error("zaten durmuş distro için stopFirst true")
	}
}

// Var olan bir dosyanın üzerine yazmak veri kaybıdır; onay metni bunu söylemeli.
func TestExportWarnsOnExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "var.tar")
	if err := os.WriteFile(path, []byte("eski yedek"), 0o600); err != nil {
		t.Fatal(err)
	}

	m := exportModel(t, wsl.Distro{Name: "alpha"})
	m.prompt = setField(m.prompt, 0, path)

	m, _ = m.submitPrompt()

	if !strings.Contains(m.confirm.act.body, "üzerine yazılacak") {
		t.Errorf("var olan dosya uyarısı yok:\n%s", m.confirm.act.body)
	}
}

func TestExportRejectsEmptyPath(t *testing.T) {
	m := exportModel(t, wsl.Distro{Name: "alpha"})
	m.prompt = setField(m.prompt, 0, "")

	m, _ = m.submitPrompt()

	if m.confirm.active {
		t.Error("boş yol ile onay diyaloğu açıldı")
	}
	if !m.prompt.active || m.prompt.err == "" {
		t.Error("form hata göstermeden kapandı")
	}
}

// Export hedefinin klasörü yoksa oluşturulmalı; kullanıcı elle mkdir yapmasın.
func TestExportCreatesTargetDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "yeni", "klasör")
	m := exportModel(t, wsl.Distro{Name: "alpha"})
	m.prompt = setField(m.prompt, 0, filepath.Join(dir, "alpha.tar"))

	m, _ = m.submitPrompt()

	if _, err := os.Stat(dir); err != nil {
		t.Errorf("hedef klasör oluşturulmadı: %v", err)
	}
	if !m.confirm.active {
		t.Error("onay diyaloğu açılmadı")
	}
}

func TestImportRejectsMissingArchive(t *testing.T) {
	m := testModel()
	m.prompt = newImportPrompt()
	m.prompt = setField(m.prompt, 0, filepath.Join(t.TempDir(), "yok.tar"))
	m.prompt = setField(m.prompt, 1, "yeni")

	m, _ = m.submitPrompt()

	if m.confirm.active {
		t.Error("olmayan arşivle onay diyaloğu açıldı")
	}
	if !strings.Contains(m.prompt.err, "bulunamadı") {
		t.Errorf("hata mesajı = %q", m.prompt.err)
	}
}

// Var olan bir distronun adıyla içeri aktarma, o distroyu bozabilir.
func TestImportRejectsDuplicateName(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "a.tar")
	if err := os.WriteFile(archive, []byte("arşiv"), 0o600); err != nil {
		t.Fatal(err)
	}

	m := testModel()
	m.distros = []wsl.Distro{{Name: "FedoraLinux-44"}}
	m.prompt = newImportPrompt()
	m.prompt = setField(m.prompt, 0, archive)
	m.prompt = setField(m.prompt, 1, "fedoralinux-44") // büyük/küçük harf farkı
	m.prompt = setField(m.prompt, 2, filepath.Join(dir, "kurulum"))

	m, _ = m.submitPrompt()

	if m.confirm.active {
		t.Error("var olan adla onay diyaloğu açıldı")
	}
	if !strings.Contains(m.prompt.err, "zaten var") {
		t.Errorf("hata mesajı = %q", m.prompt.err)
	}
}

func TestImportAcceptsValidInput(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "a.tar")
	if err := os.WriteFile(archive, []byte("arşiv"), 0o600); err != nil {
		t.Fatal(err)
	}
	install := filepath.Join(dir, "kurulum")

	m := testModel()
	m.prompt = newImportPrompt()
	m.prompt = setField(m.prompt, 0, archive)
	m.prompt = setField(m.prompt, 1, "yeni-distro")
	m.prompt = setField(m.prompt, 2, install)

	m, _ = m.submitPrompt()

	if !m.confirm.active {
		t.Fatalf("onay diyaloğu açılmadı, hata: %q", m.prompt.err)
	}
	if m.confirm.act.kind != actDistroImport {
		t.Errorf("kind = %v", m.confirm.act.kind)
	}
	if m.confirm.act.installDir != install {
		t.Errorf("installDir = %q", m.confirm.act.installDir)
	}
}

// Form açıkken tuşlar arkadaki listeye sızmamalı.
func TestPromptSwallowsKeys(t *testing.T) {
	m := testModel()
	m.distros = []wsl.Distro{{Name: "a"}, {Name: "b"}}
	m.prompt = newExportPrompt(m.distros[0])

	before := m.cursors[tabDistros]
	updated, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)

	if m.cursors[tabDistros] != before {
		t.Error("form açıkken imleç hareket etti")
	}
	if !strings.HasSuffix(m.prompt.fields[0].value, "j") {
		t.Error("yazılan harf forma gitmedi")
	}
}

func TestPromptFieldNavigation(t *testing.T) {
	p := newImportPrompt()

	p, submitted := p.update(tea.KeyMsg{Type: tea.KeyEnter})
	if submitted {
		t.Fatal("ilk alanda enter formu gönderdi")
	}
	if p.idx != 1 {
		t.Errorf("idx = %d; enter sonraki alana geçmeliydi", p.idx)
	}

	p, _ = p.update(tea.KeyMsg{Type: tea.KeyTab})
	if p.idx != 2 {
		t.Errorf("idx = %d; tab sonraki alana geçmeliydi", p.idx)
	}

	// Son alanda enter gönderir.
	if _, submitted = p.update(tea.KeyMsg{Type: tea.KeyEnter}); !submitted {
		t.Error("son alanda enter formu göndermedi")
	}
}

func TestPromptEscapeCancels(t *testing.T) {
	p, _ := newImportPrompt().update(tea.KeyMsg{Type: tea.KeyEsc})
	if p.active {
		t.Error("esc formu kapatmadı")
	}
}

func TestPromptBackspace(t *testing.T) {
	p := newImportPrompt()
	p = setField(p, 0, "abc")

	p, _ = p.update(tea.KeyMsg{Type: tea.KeyBackspace})
	if p.fields[0].value != "ab" {
		t.Errorf("değer = %q", p.fields[0].value)
	}
}

func TestHumanBytes(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{512, "512 B"},
		{2048, "2.0 KB"},
		{5 * 1024 * 1024, "5.0 MB"},
		{3 * 1024 * 1024 * 1024, "3.0 GB"},
	}
	for _, c := range cases {
		if got := humanBytes(c.in); got != c.want {
			t.Errorf("humanBytes(%d) = %q, %q bekleniyordu", c.in, got, c.want)
		}
	}
}

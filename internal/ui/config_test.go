package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/singoesdeep/wsl-extended/internal/wsl"
)

func globalConfig(content string) configModel {
	return newConfigModel(configLoadedMsg{
		target: configGlobal, subject: `C:\Users\test\.wslconfig`, content: content,
	})
}

func distroConfig(content string) configModel {
	return newConfigModel(configLoadedMsg{
		target: configDistro, subject: "FedoraLinux-44", content: content,
	})
}

// indexOfKey, form içinde bir anahtarın sırasını bulur.
func indexOfKey(c configModel, key string) int {
	for i, f := range c.fields {
		if f.Key == key {
			return i
		}
	}
	return -1
}

func TestConfigLoadsExistingValues(t *testing.T) {
	c := globalConfig("[wsl2]\nmemory=8GB\nprocessors=4\n")

	if got := c.values[indexOfKey(c, "memory")]; got != "8GB" {
		t.Errorf("memory = %q", got)
	}
	if got := c.values[indexOfKey(c, "swap")]; got != "" {
		t.Errorf("swap = %q; ayarlanmamış olmalıydı", got)
	}
}

func TestConfigEditingWritesToSelectedField(t *testing.T) {
	c := globalConfig("")
	c.idx = indexOfKey(c, "memory")

	c, _, _ = c.update(tea.KeyMsg{Type: tea.KeyEnter}) // düzenlemeye gir
	if !c.editing {
		t.Fatal("enter düzenleme kipini açmadı")
	}

	for _, r := range "8GB" {
		c, _, _ = c.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	c, _, _ = c.update(tea.KeyMsg{Type: tea.KeyEnter}) // düzenlemeden çık

	if c.editing {
		t.Error("enter düzenleme kipini kapatmadı")
	}
	if got := c.values[indexOfKey(c, "memory")]; got != "8GB" {
		t.Errorf("memory = %q", got)
	}
}

// Düzenleme kipinde j/k harf olarak yazılmalı, gezinme tuşu olarak değil.
func TestConfigEditingSwallowsNavigationKeys(t *testing.T) {
	c := globalConfig("")
	c.idx = 0
	c.editing = true

	before := c.idx
	c, _, _ = c.update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	if c.idx != before {
		t.Error("düzenleme kipinde j imleci taşıdı")
	}
	if c.values[0] != "j" {
		t.Errorf("değer = %q; harf alana yazılmalıydı", c.values[0])
	}
}

func TestConfigBackspaceClearsField(t *testing.T) {
	c := globalConfig("[wsl2]\nmemory=8GB\n")
	c.idx = indexOfKey(c, "memory")

	c, _, _ = c.update(tea.KeyMsg{Type: tea.KeyBackspace})

	if c.values[c.idx] != "" {
		t.Errorf("değer = %q; alan temizlenmeliydi", c.values[c.idx])
	}
}

// Geçersiz değer kaydedilmemeli; iş üretilmeden hata gösterilmeli.
func TestConfigSaveRejectsInvalidValue(t *testing.T) {
	c := globalConfig("")
	c.values[indexOfKey(c, "processors")] = "iki"

	c, _, save := c.save()

	if save {
		t.Fatal("geçersiz değerle kaydetme işi üretildi")
	}
	if !strings.Contains(c.err, "tam sayı") {
		t.Errorf("hata mesajı = %q", c.err)
	}
	if c.idx != indexOfKey(c, "processors") {
		t.Error("imleç hatalı alana taşınmadı")
	}
}

func TestConfigSaveWithNoChangesWarns(t *testing.T) {
	c := globalConfig("[wsl2]\nmemory=8GB\n")

	c, _, save := c.save()

	if save {
		t.Fatal("değişiklik yokken kaydetme işi üretildi")
	}
	if !strings.Contains(c.err, "Değişiklik yok") {
		t.Errorf("hata mesajı = %q", c.err)
	}
}

// Kaydetme onayı, yazılacak farkı içermeli.
func TestConfigSaveProducesDiff(t *testing.T) {
	c := globalConfig("[wsl2]\nmemory=4GB\n")
	c.values[indexOfKey(c, "memory")] = "8GB"

	_, act, save := c.save()

	if !save {
		t.Fatal("kaydetme işi üretilmedi")
	}
	if act.kind != actWriteGlobalConf {
		t.Errorf("kind = %v", act.kind)
	}
	if !strings.Contains(act.body, "- memory=4GB") || !strings.Contains(act.body, "+ memory=8GB") {
		t.Errorf("onay metni farkı göstermiyor:\n%s", act.body)
	}
	if !strings.Contains(act.content, "memory=8GB") {
		t.Errorf("yazılacak içerik yanlış:\n%s", act.content)
	}
}

// .wslconfig değişikliği ancak WSL kapatılınca etkili olur; kullanıcı bunu
// bilmezse ayarın işe yaramadığını sanır.
func TestGlobalConfigMentionsShutdown(t *testing.T) {
	c := globalConfig("[wsl2]\nmemory=4GB\n")
	c.values[indexOfKey(c, "memory")] = "8GB"

	_, act, _ := c.save()

	if !strings.Contains(act.done, "WSL'i kapat") {
		t.Errorf("shutdown uyarısı yok: %q", act.done)
	}
	if !strings.Contains(act.body, ".wslconfig.bak") {
		t.Errorf("yedek bilgisi yok:\n%s", act.body)
	}
}

func TestDistroConfigMentionsRestartAndBackup(t *testing.T) {
	c := distroConfig("[boot]\nsystemd=false\n")
	c.values[indexOfKey(c, "systemd")] = "true"

	_, act, save := c.save()

	if !save {
		t.Fatal("kaydetme işi üretilmedi")
	}
	if act.kind != actWriteDistroConf {
		t.Errorf("kind = %v", act.kind)
	}
	if act.target != "FedoraLinux-44" {
		t.Errorf("hedef = %q", act.target)
	}
	if !strings.Contains(act.done, "yeniden başlat") {
		t.Errorf("yeniden başlatma uyarısı yok: %q", act.done)
	}
	if !strings.Contains(act.body, "wsl.conf.bak") {
		t.Errorf("yedek bilgisi yok:\n%s", act.body)
	}
}

// Boş bırakılan alan dosyadan silinmeli, boş değerle yazılmamalı.
func TestConfigClearedFieldRemovesKey(t *testing.T) {
	c := globalConfig("[wsl2]\nmemory=8GB\nprocessors=4\n")
	c.values[indexOfKey(c, "memory")] = ""

	_, act, save := c.save()

	if !save {
		t.Fatal("kaydetme işi üretilmedi")
	}
	if strings.Contains(act.content, "memory") {
		t.Errorf("anahtar silinmedi:\n%s", act.content)
	}
	if !strings.Contains(act.content, "processors=4") {
		t.Errorf("diğer anahtar kayboldu:\n%s", act.content)
	}
}

// Editör açıkken tuşlar arkadaki listeye sızmamalı.
func TestConfigPanelSwallowsKeys(t *testing.T) {
	m := testModel()
	m.distros = []wsl.Distro{{Name: "a"}, {Name: "b"}}
	m.config = globalConfig("")

	before := m.cursors[tabDistros]
	updated, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = updated.(Model)

	if m.cursors[tabDistros] != before {
		t.Error("editör açıkken imleç listede hareket etti")
	}
	if !m.config.active {
		t.Error("j tuşu editörü kapattı")
	}
}

// Kaydet onaylandığında editör kapanıp onay diyaloğu açılmalı.
func TestConfigSaveOpensConfirm(t *testing.T) {
	m := testModel()
	c := globalConfig("[wsl2]\nmemory=4GB\n")
	c.values[indexOfKey(c, "memory")] = "8GB"
	m.config = c

	updated, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = updated.(Model)

	if m.config.active {
		t.Error("editör açık kaldı")
	}
	if !m.confirm.active {
		t.Fatal("onay diyaloğu açılmadı")
	}
	if m.confirm.requiresTyping() {
		t.Error("yapılandırma kaydetme ad yazdırma istemiyor olmalı")
	}
}

func TestConfigViewShowsFieldsAndUnsetMarker(t *testing.T) {
	out := globalConfig("[wsl2]\nmemory=8GB\n").view(100, 24)

	if !strings.Contains(out, "Bellek üst sınırı") {
		t.Errorf("alan etiketi yok:\n%s", out)
	}
	if !strings.Contains(out, "8GB") {
		t.Errorf("mevcut değer yok:\n%s", out)
	}
	if !strings.Contains(out, "(ayarlanmamış)") {
		t.Errorf("ayarlanmamış alan işaretlenmedi:\n%s", out)
	}
}

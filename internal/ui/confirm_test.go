package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/singoesdeep/wsl-extended/internal/wsl"
	"github.com/singoesdeep/wsl-extended/internal/wslc"
)

func runes(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// typeText, diyaloga metni harf harf yazar.
func typeText(c confirmModel, s string) (confirmModel, bool) {
	confirmed := false
	for _, r := range s {
		c, confirmed = c.update(runes(string(r)))
		if confirmed {
			break
		}
	}
	return c, confirmed
}

func dangerAction() action {
	return action{
		kind: actDistroUnregister, target: "FedoraLinux-44", display: "FedoraLinux-44",
		title: "Distroyu kalıcı olarak sil", confirmWord: "FedoraLinux-44",
	}
}

func simpleAction() action {
	return action{kind: actDistroStop, target: "Ubuntu", title: "Distroyu durdur"}
}

// Geri dönüşü olmayan işlemde enter, ad tam yazılmadan çalışmamalı.
func TestDangerConfirmRejectsEnterBeforeFullName(t *testing.T) {
	c := newConfirm(dangerAction())

	c, confirmed := typeText(c, "Fedora")
	if confirmed {
		t.Fatal("yarım ad yazılmışken onay verildi")
	}

	_, confirmed = c.update(tea.KeyMsg{Type: tea.KeyEnter})
	if confirmed {
		t.Fatal("ad tamamlanmadan enter onayladı")
	}
}

func TestDangerConfirmRejectsWrongName(t *testing.T) {
	c := newConfirm(dangerAction())

	c, _ = typeText(c, "fedoralinux-44") // küçük harf: eşleşmemeli
	if c.satisfied() {
		t.Fatal("büyük/küçük harf farkı kabul edildi")
	}

	if _, confirmed := c.update(tea.KeyMsg{Type: tea.KeyEnter}); confirmed {
		t.Fatal("yanlış ad ile enter onayladı")
	}
}

func TestDangerConfirmAcceptsExactName(t *testing.T) {
	c := newConfirm(dangerAction())

	c, _ = typeText(c, "FedoraLinux-44")
	if !c.satisfied() {
		t.Fatal("doğru ad yazıldı ama satisfied false")
	}

	if _, confirmed := c.update(tea.KeyMsg{Type: tea.KeyEnter}); !confirmed {
		t.Fatal("doğru ad ile enter onaylamadı")
	}
}

// Tehlikeli kipte "y" bir onay tuşu değil, yazılan metnin parçasıdır.
func TestDangerConfirmTreatsYAsInput(t *testing.T) {
	c := newConfirm(dangerAction())

	c, confirmed := c.update(runes("y"))
	if confirmed {
		t.Fatal("tehlikeli kipte y tuşu işlemi onayladı")
	}
	if c.input != "y" {
		t.Errorf("input = %q; y metne eklenmeliydi", c.input)
	}
}

func TestDangerConfirmBackspace(t *testing.T) {
	c := newConfirm(dangerAction())
	c, _ = typeText(c, "Fedorx")

	c, _ = c.update(tea.KeyMsg{Type: tea.KeyBackspace})
	if c.input != "Fedor" {
		t.Errorf("input = %q; backspace son harfi silmeliydi", c.input)
	}
}

func TestSimpleConfirmYesNo(t *testing.T) {
	if _, confirmed := newConfirm(simpleAction()).update(runes("y")); !confirmed {
		t.Error("y onaylamadı")
	}
	if _, confirmed := newConfirm(simpleAction()).update(runes("n")); confirmed {
		t.Error("n onayladı")
	}
	if _, confirmed := newConfirm(simpleAction()).update(runes("e")); !confirmed {
		t.Error("e (evet) onaylamadı")
	}
}

func TestConfirmEscapeCancels(t *testing.T) {
	for _, a := range []action{simpleAction(), dangerAction()} {
		c, confirmed := newConfirm(a).update(tea.KeyMsg{Type: tea.KeyEsc})
		if confirmed {
			t.Errorf("%s: esc onayladı", a.title)
		}
		if c.active {
			t.Errorf("%s: esc diyalogu kapatmadı", a.title)
		}
	}
}

// Ctrl+C diyalogu kapatır ama işlemi çalıştırmaz.
func TestConfirmCtrlCDoesNotRun(t *testing.T) {
	c := newConfirm(dangerAction())
	c, _ = typeText(c, "FedoraLinux-44")

	c, confirmed := c.update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if confirmed {
		t.Fatal("ctrl+c işlemi çalıştırdı")
	}
	if c.active {
		t.Error("ctrl+c diyalogu kapatmadı")
	}
}

func TestConfirmViewShowsExpectedName(t *testing.T) {
	out := newConfirm(dangerAction()).view(100)
	if !strings.Contains(out, "FedoraLinux-44") {
		t.Errorf("diyalogda yazılması gereken ad görünmüyor:\n%s", out)
	}
}

// Silme tuşu, seçili satır için doğru hedefi ve doğru onay kipini üretmeli.
func TestActionForSelectsCorrectTarget(t *testing.T) {
	m := testModel()
	m.distros = []wsl.Distro{{Name: "alpha"}, {Name: "beta"}}
	m.cursors[tabDistros] = 1

	act, ok := m.actionFor("d")
	if !ok {
		t.Fatal("d tuşu için işlem üretilmedi")
	}
	if act.target != "beta" {
		t.Errorf("hedef = %q; imlecin üstündeki satır olmalıydı", act.target)
	}
	if act.confirmWord != "beta" {
		t.Errorf("confirmWord = %q; unregister ad yazdırmalı", act.confirmWord)
	}
	if act.kind != actDistroUnregister {
		t.Errorf("kind = %v", act.kind)
	}
}

// Birim silme veri kaybıdır: ad yazdırma kipinde olmalı.
func TestVolumeRemoveRequiresTyping(t *testing.T) {
	m := testModel()
	m.active = tabVolumes
	m.volumes = []wslc.Volume{{Name: "data"}}

	act, ok := m.actionFor("d")
	if !ok {
		t.Fatal("işlem üretilmedi")
	}
	if act.confirmWord != "data" {
		t.Errorf("confirmWord = %q; birim silme ad yazdırmalı", act.confirmWord)
	}
}

// Kapsayıcı silme geri alınabilir sayılır: basit y/n yeterli.
func TestContainerRemoveIsSimpleConfirm(t *testing.T) {
	m := testModel()
	m.active = tabContainers
	m.containers = []wslc.Container{{ID: "abc", Names: "web"}}

	act, ok := m.actionFor("d")
	if !ok {
		t.Fatal("işlem üretilmedi")
	}
	if act.confirmWord != "" {
		t.Errorf("confirmWord = %q; kapsayıcı silme y/n olmalı", act.confirmWord)
	}
	if act.target != "web" {
		t.Errorf("hedef = %q", act.target)
	}
}

// s tuşu tek başına duruma göre davranmalı: duran başlar, çalışan durur.
func TestStartStopToggleForDistro(t *testing.T) {
	m := testModel()

	m.distros = []wsl.Distro{{Name: "a", State: wsl.StateStopped}}
	act, ok := m.actionFor("s")
	if !ok || act.kind != actDistroStart {
		t.Errorf("durmuş distroda s başlatmalıydı, kind = %v", act.kind)
	}

	m.distros = []wsl.Distro{{Name: "a", State: wsl.StateRunning}}
	act, ok = m.actionFor("s")
	if !ok || act.kind != actDistroStop {
		t.Errorf("çalışan distroda s durdurmalıydı, kind = %v", act.kind)
	}
}

func TestStartStopToggleForContainer(t *testing.T) {
	m := testModel()
	m.active = tabContainers

	m.containers = []wslc.Container{{Names: "web", State: "exited"}}
	act, ok := m.actionFor("s")
	if !ok || act.kind != actContainerStart {
		t.Errorf("durmuş kapsayıcıda s başlatmalıydı, kind = %v", act.kind)
	}

	m.containers = []wslc.Container{{Names: "web", State: "running"}}
	act, ok = m.actionFor("s")
	if !ok || act.kind != actContainerStop {
		t.Errorf("çalışan kapsayıcıda s durdurmalıydı, kind = %v", act.kind)
	}
}

// Boş listede aksiyon tuşu hiçbir şey yapmamalı.
func TestActionOnEmptyListIsNoop(t *testing.T) {
	m := testModel()
	for _, key := range []string{"s", "S", "d", "u", "K"} {
		if _, ok := m.actionFor(key); ok {
			t.Errorf("boş listede %q tuşu işlem üretti", key)
		}
	}
}

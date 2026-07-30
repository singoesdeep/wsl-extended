package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/singoesdeep/wsl-extended/internal/store"
	"github.com/singoesdeep/wsl-extended/internal/wsl"
	"github.com/singoesdeep/wsl-extended/internal/wslconf"
)

// newExportPrompt, seçili distro için hedef dosya soran formu kurar.
func newExportPrompt(d wsl.Distro) promptModel {
	body := "Distro tek bir arşiv dosyasına yazılacak."
	if d.IsRunning() {
		// Çalışan bir distronun dosya sistemi export sırasında değişebilir;
		// tutarlı bir yedek için önce durdurulur.
		body += "\n" + d.Name + " şu anda çalışıyor; tutarlı bir yedek için önce durdurulacak."
	}
	body += "\n\nUzantı biçimi belirler: .tar, .tar.gz ya da .vhdx"

	return promptModel{
		active: true, kind: promptExport, subject: d.Name,
		title: "Distroyu dışa aktar · " + d.Name,
		body:  body,
		fields: []promptField{
			{label: "Hedef dosya", value: defaultExportPath(d.Name), hint: "C:\\yedek\\distro.tar"},
		},
	}
}

func defaultExportPath(name string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	file := fmt.Sprintf("%s-%s.tar", name, time.Now().Format("2006-01-02"))
	return filepath.Join(home, file)
}

// newImportPrompt, arşivden yeni distro oluşturan formu kurar. Aynı form
// klonlama için de kullanılır: önce export alınır, sonra arşiv başka bir adla
// içeri aktarılır.
func newImportPrompt() promptModel {
	return promptModel{
		active: true, kind: promptImport,
		title: "Arşivden distro oluştur",
		body: "Var olan bir arşivden yeni distro kaydedilir.\n" +
			"Klonlamak için önce e ile yedek al, sonra burada başka bir ad ver.",
		fields: []promptField{
			{label: "Arşiv dosyası", hint: "C:\\yedek\\distro.tar"},
			{label: "Yeni distro adı", hint: "Fedora-kopya"},
			{label: "Kurulum dizini", hint: defaultInstallDir("<ad>")},
		},
	}
}

func defaultInstallDir(name string) string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		base = home
	}
	return filepath.Join(base, "WSL", name)
}

// newAliasPrompt, distroya görünen ad veren formu kurar.
//
// Bu ad yalnızca bu uygulamanın verisinde saklanır; WSL'deki gerçek distro adı
// değişmez ve tüm komutlar gerçek adla çalışmaya devam eder.
func newAliasPrompt(realName, current string) promptModel {
	return promptModel{
		active: true, kind: promptAlias, subject: realName,
		title: "Görünen adı değiştir · " + realName,
		body: "Bu ad yalnızca bu uygulamada görünür; WSL'deki gerçek ad değişmez\n" +
			"ve komutlar gerçek adla çalışır. Boş bırakırsan gerçek ada döner.",
		fields: []promptField{
			{label: "Görünen ad", value: current, hint: realName},
		},
	}
}

// submitPrompt, formu doğrular ve geçerliyse işi üretir. Doğrulama başarısızsa
// form açık kalır ve hatayı gösterir.
func (m Model) submitPrompt() (Model, tea.Cmd) {
	values := m.prompt.values()

	switch m.prompt.kind {
	case promptAlias:
		real := m.prompt.subject
		m.data.SetAlias(real, values[0])

		m.prompt = promptModel{}
		m.noticeAt = time.Now()
		if err := store.Save(m.dataPath, m.data); err != nil {
			m.notice, m.noticeErr = "Görünen ad kaydedilemedi: "+err.Error(), true
			return m, nil
		}
		if m.data.HasAlias(real) {
			m.notice, m.noticeErr = real+" artık "+m.data.Alias(real)+" olarak görünüyor", false
		} else {
			m.notice, m.noticeErr = real+" gerçek adına döndü", false
		}
		return m, nil

	case promptResize:
		size := values[0]
		if size == "" {
			m.prompt.err = "Boyut boş olamaz."
			return m, nil
		}
		if err := wslconf.CheckSize(size); err != nil {
			m.prompt.err = err.Error()
			return m, nil
		}

		name := m.prompt.subject
		display := m.displayName(name)
		m.prompt = promptModel{}
		m.confirm = newConfirm(action{
			kind: actDistroResize, target: name, display: display, size: size,
			title: "Diski büyüt",
			body: display + " diski " + size + " boyutuna getirilecek.\n" +
				"Distro çalışıyorsa işlem başarısız olur; önce durdurmalısın.",
			done: display + " diski " + size + " oldu",
		})
		return m, nil

	case promptMove:
		dir := values[0]
		if dir == "" {
			m.prompt.err = "Hedef klasör boş olamaz."
			return m, nil
		}

		name := m.prompt.subject
		display := m.displayName(name)
		m.prompt = promptModel{}
		m.confirm = newConfirm(action{
			kind: actDistroMove, target: name, display: display, path: dir,
			title: "Başka konuma taşı",
			body: display + " şuraya taşınacak:\n" + dir +
				"\n\nDistro çalışıyorsa işlem başarısız olur; önce durdurmalısın.",
			done: display + " taşındı",
		})
		return m, nil

	case promptExport:
		path := values[0]
		if path == "" {
			m.prompt.err = "Hedef dosya boş olamaz."
			return m, nil
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			m.prompt.err = "Hedef klasör oluşturulamadı: " + err.Error()
			return m, nil
		}

		d, ok := m.selectedDistro()
		if !ok {
			m.prompt = promptModel{}
			return m, nil
		}

		act := action{
			kind: actDistroExport, target: d.Name, display: d.Name,
			path: path, stopFirst: d.IsRunning(),
			title: "Distroyu dışa aktar",
			body:  d.Name + " şuraya yazılacak:\n" + path,
			done:  d.Name + " dışa aktarıldı: " + path,
		}
		if d.IsRunning() {
			act.body += "\n\nDistro önce durdurulacak."
		}
		// Var olan bir dosyanın üzerine yazmak veri kaybıdır; bu durumda onay
		// metni de değişir.
		if _, err := os.Stat(path); err == nil {
			act.body += "\n\nBu dosya zaten var ve üzerine yazılacak."
		}

		m.prompt = promptModel{}
		m.confirm = newConfirm(act)
		return m, nil

	case promptImport:
		archive, name, dir := values[0], values[1], values[2]
		if archive == "" || name == "" {
			m.prompt.err = "Arşiv dosyası ve distro adı zorunlu."
			return m, nil
		}
		if _, err := os.Stat(archive); err != nil {
			m.prompt.err = "Arşiv bulunamadı: " + archive
			return m, nil
		}
		for _, d := range m.distros {
			if strings.EqualFold(d.Name, name) {
				m.prompt.err = name + " adında bir distro zaten var."
				return m, nil
			}
		}
		if dir == "" {
			dir = defaultInstallDir(name)
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			m.prompt.err = "Kurulum dizini oluşturulamadı: " + err.Error()
			return m, nil
		}

		act := action{
			kind: actDistroImport, target: name, display: name,
			path: archive, installDir: dir,
			title: "Arşivden distro oluştur",
			body:  archive + "\n" + name + " olarak şuraya kurulacak:\n" + dir,
			done:  name + " oluşturuldu",
		}

		m.prompt = promptModel{}
		m.confirm = newConfirm(act)
		return m, nil
	}

	m.prompt = promptModel{}
	return m, nil
}

// humanBytes, ilerleme göstergesi için okunabilir boyut üretir.
func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit && exp < 3; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGT"[exp])
}

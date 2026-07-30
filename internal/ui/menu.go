package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/singoesdeep/wsl-extended/internal/ui/theme"
)

// menuItem, menüdeki tek bir seçenek.
type menuItem struct {
	id    string
	label string
	desc  string
}

type menuKind int

const (
	menuDisk menuKind = iota
	menuOpen
	menuSettings
	menuContainer
)

// menuModel, birkaç seçenek arasından seçim yaptıran küçük panodur. Her işlem
// için ayrı tuş atamak yerine ilgili işlemleri tek yerde toplar; böylece tuş
// haritası küçük ve küçük harfli kalır.
type menuModel struct {
	active  bool
	kind    menuKind
	title   string
	subject string
	items   []menuItem
	idx     int
}

// update, menü açıkken tuşları işler. Dönen id boş değilse seçim yapılmıştır.
func (m menuModel) update(msg tea.KeyMsg) (menuModel, string) {
	switch msg.String() {
	case "esc", "q":
		return menuModel{}, ""

	case "down", "j":
		m.idx = min(m.idx+1, len(m.items)-1)

	case "up", "k":
		m.idx = max(m.idx-1, 0)

	case "enter":
		id := m.items[m.idx].id
		return menuModel{}, id
	}
	return m, ""
}

func (m menuModel) view(width int) string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(theme.DialogTitle.Render(m.title))
	if m.subject != "" {
		b.WriteString(theme.DialogHint.Render(m.subject))
	}
	b.WriteString("\n")
	b.WriteString(rule(width))
	b.WriteString("\n")

	for i, it := range m.items {
		mark := "  "
		label := it.label
		if i == m.idx {
			mark = theme.Mark.Render(theme.SelectionMark) + " "
			label = theme.RowSelected.Render(label)
		}

		b.WriteString("\n " + mark + label)
		if it.desc != "" {
			b.WriteString("\n     " + theme.DialogHint.Render(it.desc))
		}
	}

	return b.String()
}

// diskMenu, distronun disk işlemlerini toplayan menüyü kurar.
func diskMenu(display string, running bool) menuModel {
	warn := ""
	if running {
		// resize ve move, distro kapalıyken çalışır; kullanıcı bunu önceden
		// bilmezse komut hata verir ve nedeni belirsiz kalır.
		warn = " (distro çalışıyor; önce s ile durdur)"
	}

	return menuModel{
		active: true, kind: menuDisk, title: "Disk işlemleri", subject: display,
		items: []menuItem{
			{id: "resize", label: "Diski büyüt",
				desc: "Sanal diskin üst sınırını değiştirir" + warn},
			{id: "sparse-on", label: "Seyrek diski aç",
				desc: "Silinen dosyaların yeri Windows tarafında geri kazanılır"},
			{id: "sparse-off", label: "Seyrek diski kapat",
				desc: "Disk sabit boyutta kalır"},
			{id: "move", label: "Başka konuma taşı",
				desc: "Distronun dosyalarını başka bir klasöre taşır" + warn},
		},
	}
}

// openMenu, distroyu Windows araçlarıyla açma seçenekleri.
func openMenu(display string) menuModel {
	return menuModel{
		active: true, kind: menuOpen, title: "Şununla aç", subject: display,
		items: []menuItem{
			{id: "explorer", label: "Windows Gezgini",
				desc: `Distronun dosya sistemini \\wsl.localhost üzerinden açar`},
			{id: "vscode", label: "VS Code",
				desc: "code --remote ile distroya bağlanır (VS Code kurulu olmalı)"},
			{id: "terminal", label: "Yeni pencerede kabuk",
				desc: "Distroyu ayrı bir konsol penceresinde açar"},
		},
	}
}

// settingsMenu, yapılandırma ve WSL geneli işlemler.
func settingsMenu(hasDistro bool) menuModel {
	items := []menuItem{
		{id: "wslconfig", label: ".wslconfig",
			desc: "Bellek, işlemci, takas, ağ kipi — tüm WSL2 için geçerli"},
	}
	if hasDistro {
		items = append(items, menuItem{id: "wslconf", label: "Seçili distronun wsl.conf'u",
			desc: "systemd, varsayılan kullanıcı, automount — okumak distroyu başlatır"})
	}
	items = append(items,
		menuItem{id: "shutdown", label: "WSL'i kapat",
			desc: ".wslconfig değişikliklerinin etkili olması için gerekir"},
		menuItem{id: "update", label: "WSL'i güncelle",
			desc: "wsl --update çalıştırır"},
	)

	return menuModel{active: true, kind: menuSettings, title: "Ayarlar", items: items}
}

// containerMenu, kapsayıcı oluşturma ve imaj çekme.
func containerMenu() menuModel {
	return menuModel{
		active: true, kind: menuContainer, title: "Kapsayıcı işlemleri",
		items: []menuItem{
			{id: "pull", label: "İmaj çek",
				desc: "wslc pull ile bir imaj indirir"},
			{id: "run", label: "Kapsayıcı çalıştır",
				desc: "İmaj, ad, port ve birim vererek yeni kapsayıcı başlatır"},
		},
	}
}

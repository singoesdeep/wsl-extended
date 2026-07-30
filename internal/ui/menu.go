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

// menuModel, birkaç seçenek arasından seçim yaptıran küçük panodur. Her işlem
// için ayrı tuş atamak yerine ilgili işlemleri tek yerde toplar.
type menuModel struct {
	active  bool
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
		active: true, title: "Disk işlemleri", subject: display,
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

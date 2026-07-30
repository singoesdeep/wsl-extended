package ui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// selectedOnline, mağazada seçili dağıtımı döndürür.
func (m Model) selectedOnline() (onlineRow, bool) {
	i := m.cursors[tabStore]
	if i < 0 || i >= len(m.online) {
		return onlineRow{}, false
	}
	o := m.online[i]
	return onlineRow{name: o.Name, friendly: o.Friendly,
		installed: m.installedNames()[strings.ToLower(o.Name)]}, true
}

type onlineRow struct {
	name      string
	friendly  string
	installed bool
}

// confirmInstall, mağazadan seçili dağıtımın kurulumunu onaya sunar.
func (m Model) confirmInstall() (tea.Model, tea.Cmd) {
	row, ok := m.selectedOnline()
	if !ok {
		return m, nil
	}
	if row.installed {
		m.notice, m.noticeErr = row.name+" zaten kurulu", true
		m.noticeAt = time.Now()
		return m, nil
	}

	m.confirm = newConfirm(action{
		kind: actDistroInstall, target: row.name, display: row.friendly,
		title: "Dağıtımı kur",
		body: row.friendly + " (" + row.name + ") indirilip kurulacak.\n" +
			"İndirme boyutuna göre birkaç dakika sürebilir; ilerleme ekranda akar.\n\n" +
			"Kurulumdan sonra dağıtım otomatik açılmaz; kullanıcı hesabını\n" +
			"ilk kez enter ile kabuğa girdiğinde oluşturursun.",
		done: row.name + " kuruldu",
	})
	return m, nil
}

// handleDiskChoice, disk menüsündeki seçimi ilgili forma ya da onaya bağlar.
func (m Model) handleDiskChoice(choice string) (tea.Model, tea.Cmd) {
	d, ok := m.selectedDistro()
	if !ok {
		return m, nil
	}
	display := m.displayName(d.Name)

	switch choice {
	case "resize":
		m.prompt = promptModel{
			active: true, kind: promptResize, subject: d.Name,
			title: "Diski büyüt · " + display,
			body: "Sanal diskin üst sınırı değiştirilir. Küçültme desteklenmez;\n" +
				"yalnızca mevcut boyuttan büyük bir değer verilebilir.",
			fields: []promptField{
				{label: "Yeni boyut", hint: "60GB"},
			},
		}

	case "move":
		m.prompt = promptModel{
			active: true, kind: promptMove, subject: d.Name,
			title: "Başka konuma taşı · " + display,
			body:  "Distronun dosyaları belirtilen klasöre taşınır.",
			fields: []promptField{
				{label: "Hedef klasör", hint: `D:\wsl\` + d.Name},
			},
		}

	case "sparse-on", "sparse-off":
		on := choice == "sparse-on"
		title, body, done := "Seyrek diski aç", "", ""
		if on {
			body = display + " için seyrek disk açılacak.\n" +
				"Silinen dosyaların yeri Windows tarafında otomatik geri kazanılır."
			done = display + " için seyrek disk açıldı"
		} else {
			title = "Seyrek diski kapat"
			body = display + " için seyrek disk kapatılacak.\n" +
				"Disk dosyası sabit boyutta kalır."
			done = display + " için seyrek disk kapatıldı"
		}

		m.confirm = newConfirm(action{
			kind: actDistroSparse, target: d.Name, display: display,
			sparse: on, title: title, body: body, done: done,
		})
	}

	return m, nil
}

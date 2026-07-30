package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/singoesdeep/wsl-extended/internal/wsl"
)

// handleMenuChoice, açık menüdeki seçimi ilgili işe yönlendirir.
func (m Model) handleMenuChoice(kind menuKind, choice string) (tea.Model, tea.Cmd) {
	switch kind {
	case menuDisk:
		return m.handleDiskChoice(choice)
	case menuOpen:
		return m.handleOpenChoice(choice)
	case menuSettings:
		return m.handleSettingsChoice(choice)
	case menuContainer:
		return m.handleContainerChoice(choice)
	}
	return m, nil
}

// handleOpenChoice, distroyu Windows araçlarıyla açar.
//
// Gezgin ve VS Code ayrı pencerelerde açıldığı için terminal devredilmez;
// yalnızca kabuk, terminali gerçekten kullandığından tea.ExecProcess ister.
func (m Model) handleOpenChoice(choice string) (tea.Model, tea.Cmd) {
	d, ok := m.selectedDistro()
	if !ok {
		return m, nil
	}

	switch choice {
	case "explorer":
		if err := wsl.OpenExplorer(d.Name); err != nil {
			m.notice, m.noticeErr, m.noticeAt = err.Error(), true, time.Now()
		} else {
			m.notice, m.noticeErr, m.noticeAt = "Gezgin açıldı", false, time.Now()
		}
		return m, nil

	case "vscode":
		if err := wsl.OpenVSCode(d.Name); err != nil {
			// VS Code kurulu değilse komut bulunamaz; bunu açıkça söylemek
			// gerekir, aksi hâlde hiçbir şey olmamış gibi görünür.
			m.notice, m.noticeErr, m.noticeAt =
				"VS Code açılamadı: "+err.Error(), true, time.Now()
		} else {
			m.notice, m.noticeErr, m.noticeAt = "VS Code açılıyor", false, time.Now()
		}
		return m, nil

	case "terminal":
		if err := wsl.OpenWindow(d.Name); err != nil {
			m.notice, m.noticeErr, m.noticeAt = err.Error(), true, time.Now()
		}
		return m, nil
	}
	return m, nil
}

func (m Model) handleSettingsChoice(choice string) (tea.Model, tea.Cmd) {
	switch choice {
	case "wslconfig":
		return m, loadGlobalConfig()

	case "wslconf":
		d, ok := m.selectedDistro()
		if !ok {
			return m, nil
		}
		m.busy, m.busyLabel = true, "wsl.conf okunuyor ("+m.displayName(d.Name)+" başlatılıyor)"
		return m, loadDistroConfig(d.Name)

	case "shutdown":
		m.confirm = newConfirm(action{
			kind:  actWSLShutdown,
			title: "WSL'i kapat",
			body: "Tüm distrolar durdurulacak ve WSL sanal makinesi kapanacak.\n" +
				"Çalışan işler sonlanır. Veri kaybı yok.",
			done: "WSL kapatıldı",
		})
		return m, nil

	case "update":
		m.confirm = newConfirm(action{
			kind:  actWSLUpdate,
			title: "WSL'i güncelle",
			body:  "wsl --update çalıştırılacak. Yeni sürüm varsa indirilir.",
			done:  "WSL güncellendi",
		})
		return m, nil
	}
	return m, nil
}

func (m Model) handleContainerChoice(choice string) (tea.Model, tea.Cmd) {
	switch choice {
	case "pull":
		m.prompt = promptModel{
			active: true, kind: promptPull,
			title: "İmaj çek",
			body:  "İndirilecek imajın adını yaz. İlerleme ayrı bir ekranda görünür.",
			fields: []promptField{
				{label: "İmaj", hint: "docker.io/library/alpine:latest"},
			},
		}

	case "run":
		m.prompt = promptModel{
			active: true, kind: promptRun,
			title: "Kapsayıcı çalıştır",
			body: "Kapsayıcı arka planda başlatılır.\n" +
				"Boş bırakılan alanlar komuta eklenmez.",
			fields: []promptField{
				{label: "İmaj", hint: "alpine:latest"},
				{label: "Ad (isteğe bağlı)", hint: "web"},
				{label: "Port eşlemesi (isteğe bağlı)", hint: "8080:80"},
				{label: "Birim (isteğe bağlı)", hint: "data:/veri"},
				{label: "Komut (isteğe bağlı)", hint: "sh"},
			},
		}
	}
	return m, nil
}

package ui

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/singoesdeep/wsl-extended/internal/wsl"
)

// onlineMsg, kurulabilir dağıtım kataloğunun sonucudur.
type onlineMsg struct {
	items []wsl.OnlineDistro
	err   error
}

// loadOnline, kurulabilir dağıtımları çeker. Ağ üzerinden yapıldığı için
// listeleme komutlarından daha uzun sürebilir.
func loadOnline() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		items, err := wsl.ListOnline(ctx)
		return onlineMsg{items, err}
	}
}

// installedNames, kurulu distro adlarını küçük harfe indirgenmiş kümede verir.
func (m Model) installedNames() map[string]bool {
	set := make(map[string]bool, len(m.distros))
	for _, d := range m.distros {
		set[strings.ToLower(d.Name)] = true
	}
	return set
}

// runInstall, kurulumu terminali devrederek çalıştırır.
//
// Çıktıyı boruya almak yerine terminal devredilir; wsl.exe indirme yüzdesini
// yalnızca gerçek konsola bağlıyken çizer. Bubble Tea alternatif ekranı bırakır,
// kurulum kendi ilerleme çubuğunu gösterir ve bitince arayüz geri gelir.
func runInstall(name string) tea.Cmd {
	return tea.ExecProcess(wsl.InstallCommand(name), func(err error) tea.Msg {
		return installDoneMsg{name: name, err: err}
	})
}

type installDoneMsg struct {
	name string
	err  error
}

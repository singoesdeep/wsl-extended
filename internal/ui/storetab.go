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

// startInstall, dağıtım kurulumunu başlatır ve çıktısını canlı panele bağlar.
func startInstall(name string) (logModel, tea.Cmd) {
	ctx, cancel := context.WithCancel(context.Background())

	ch, err := wsl.StreamInstall(ctx, name)
	if err != nil {
		cancel()
		return logModel{active: true, target: name, err: err}, nil
	}
	return newStreamPanel("kurulum · "+name, ch, cancel)
}

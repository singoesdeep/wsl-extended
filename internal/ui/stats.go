package ui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/singoesdeep/wsl-extended/internal/ui/theme"
	"github.com/singoesdeep/wsl-extended/internal/wslc"
)

// statsModel, kapsayıcıların kaynak kullanımını gösteren paneldir.
//
// `wslc stats` akış yapmadığı için panel, etkin olduğu sürece düzenli aralıkla
// yeniden çağrılır.
type statsModel struct {
	active bool
	items  []wslc.Stat
	err    error
}

type statsMsg struct {
	items []wslc.Stat
	err   error
}

func loadStats() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
		defer cancel()

		items, err := wslc.Stats(ctx)
		return statsMsg{items, err}
	}
}

func (m statsModel) view(width, height int) string {
	if m.err != nil {
		return theme.Error.Render("İstatistikler alınamadı: " + m.err.Error())
	}

	head := theme.DialogTitle.Render("Kaynak kullanımı") +
		theme.DialogHint.Render("  (esc kapat)")

	if len(m.items) == 0 {
		return head + "\n" + theme.Empty.Render("Çalışan kapsayıcı yok.")
	}

	cols := []column{
		{title: "NAME"},
		{title: "CPU %", width: 10},
		{title: "MEM", width: 20},
		{title: "MEM %", width: 8},
		{title: "NET I/O", width: 18},
		{title: "BLOCK I/O", width: 18},
		{title: "PIDS", width: 6},
	}

	rows := make([][]string, 0, len(m.items))
	for _, s := range m.items {
		name := s.Name.String()
		if name == "" {
			name = s.ID.String()
		}
		rows = append(rows, []string{
			name, s.CPUPerc.String(), s.MemUsage.String(), s.MemPerc.String(),
			s.NetIO.String(), s.BlockIO.String(), s.PIDs.String(),
		})
	}

	// İmleç -1: bu panelde seçim yok, yalnızca izleme var.
	return head + "\n" + renderTable(cols, rows, -1, width, height-1)
}

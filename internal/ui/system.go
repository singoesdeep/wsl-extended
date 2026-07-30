package ui

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/singoesdeep/wsl-extended/internal/ui/theme"
	"github.com/singoesdeep/wsl-extended/internal/wsl"
)

// systemModel, WSL geneli durumu gösteren paneldir.
type systemModel struct {
	active bool

	status  string
	version string

	total     int
	running   int
	diskTotal int64

	err error
}

type systemMsg struct {
	status    string
	version   string
	total     int
	running   int
	diskTotal int64
	err       error
}

// loadSystem, WSL durumunu ve kurulu distroların toplam disk kullanımını
// toplar. Disk boyutları registry'den okunur; hiçbir distro başlatılmaz.
func loadSystem() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
		defer cancel()

		status, err := wsl.Status(ctx)
		if err != nil {
			return systemMsg{err: err}
		}
		version, _ := wsl.Version(ctx)

		ds, err := wsl.List(ctx)
		if err != nil {
			return systemMsg{status: status, version: version, err: err}
		}

		msg := systemMsg{status: status, version: version, total: len(ds)}
		for _, d := range ds {
			if d.IsRunning() {
				msg.running++
			}
			if info, err := wsl.Describe(ctx, wsl.Distro{Name: d.Name, State: wsl.StateStopped}); err == nil {
				msg.diskTotal += info.DiskSize
			}
		}
		return msg
	}
}

func (m systemModel) view(width, height int) string {
	if m.err != nil {
		return theme.Error.Render("Sistem durumu alınamadı: " + m.err.Error())
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(theme.DialogTitle.Render("WSL sistem durumu"))
	b.WriteString("\n")
	b.WriteString(rule(width))
	b.WriteString("\n")

	row := func(label, value string) {
		if strings.TrimSpace(value) == "" {
			return
		}
		b.WriteString("\n  " + theme.Label.Render(fitCell(label, 20)) + "  " +
			theme.Value.Render(value))
	}

	row("Sürüm", m.version)
	row("Kurulu distro", itoa(m.total))
	row("Çalışan", itoa(m.running))
	if m.diskTotal > 0 {
		row("Toplam disk", humanBytes(m.diskTotal))
	}

	if s := strings.TrimSpace(m.status); s != "" {
		b.WriteString("\n\n")
		b.WriteString(theme.DialogHint.Render("wsl --status"))
		b.WriteString("\n")
		for _, line := range strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n") {
			b.WriteString("\n  " + theme.Value.Render(fitCell(line, max(10, width-4))))
		}
	}

	return b.String()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

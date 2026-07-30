package ui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/singoesdeep/wsl-extended/internal/ui/theme"
	"github.com/singoesdeep/wsl-extended/internal/wsl"
)

// inspectModel, seçili distronun ayrıntılarını gösteren paneldir.
type inspectModel struct {
	active bool
	info   wsl.Info
	alias  string
	err    error
}

type inspectMsg struct {
	info wsl.Info
	err  error
}

// loadInspect, distro bilgilerini toplar. Distro çalışmıyorsa yalnızca statik
// bilgiler okunur; bilgi almak için distro başlatılmaz.
func loadInspect(d wsl.Distro) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
		defer cancel()

		info, err := wsl.Describe(ctx, d)
		return inspectMsg{info, err}
	}
}

func (m inspectModel) view(width, height int) string {
	if m.err != nil {
		return theme.Error.Render("Bilgi alınamadı: " + m.err.Error())
	}

	i := m.info

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(theme.DialogTitle.Render(i.Name))
	if m.alias != "" && m.alias != i.Name {
		b.WriteString(theme.DialogHint.Render("görünen ad: " + m.alias))
	}
	b.WriteString("\n")
	b.WriteString(rule(width))
	b.WriteString("\n")

	row := func(label, value string) {
		if strings.TrimSpace(value) == "" {
			return
		}
		b.WriteString("\n  " + theme.Label.Render(fitCell(label, 18)) + "  " +
			theme.Value.Render(value))
	}

	state := string(i.State)
	if i.State == wsl.StateRunning {
		state = theme.StateDot(true) + " " + state
	} else {
		state = theme.StateDot(false) + " " + state
	}

	row("Durum", state)
	row("WSL sürümü", i.Version)
	if i.Default {
		row("Varsayılan", "evet")
	}
	row("Varsayılan UID", fmt.Sprintf("%d", i.DefaultUID))
	row("Kimlik", i.GUID)

	b.WriteString("\n")
	row("Kurulum dizini", i.BasePath)
	row("Disk dosyası", i.DiskPath)
	if i.DiskSize > 0 {
		row("Disk boyutu", humanBytes(i.DiskSize))
	}

	b.WriteString("\n")
	switch {
	case i.Live:
		row("Çekirdek", i.Kernel)
		row("IP adresi", i.IP)
		if i.DiskUsed != "" {
			row("Kök disk", i.DiskUsed+" kullanılıyor · "+i.DiskFree+" boş · "+i.DiskUse)
		}

	case i.State == wsl.StateRunning:
		// Distro çalışıyor ama canlı bilgi alınamadı; "çalışmıyor" demek
		// yanıltıcı olurdu.
		msg := "Distro çalışıyor ama canlı bilgi alınamadı."
		if i.LiveErr != nil {
			msg += "\n  " + i.LiveErr.Error()
		}
		b.WriteString("\n  " + theme.DialogHint.Render(msg))

	default:
		b.WriteString("\n  " + theme.DialogHint.Render(
			"Distro çalışmıyor; çekirdek, IP ve disk kullanımı için s ile başlat."))
	}

	return b.String()
}

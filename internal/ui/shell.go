package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/singoesdeep/wsl-extended/internal/wsl"
	"github.com/singoesdeep/wsl-extended/internal/wslc"
)

type shellDoneMsg struct{ err error }

// openShell, seçili distro ya da kapsayıcı için etkileşimli kabuk açar.
//
// Komut tea.ExecProcess ile çalıştırılır: Bubble Tea alternatif ekranı bırakır,
// terminali komuta devreder ve komut bitince arayüzü geri yükler. Aynı komutu
// doğrudan exec.Command ile çalıştırmak terminali bozardı — iki program aynı
// anda aynı ekranı yönetmeye çalışırdı.
//
// Windows'ta TUI içine gömülü bir terminal (pty) hedeflenmiyor; ConPTY işi
// orantısız karmaşıklaştırır ve devretme yaklaşımı pratikte aynı sonucu verir.
func (m Model) openShell() (tea.Model, tea.Cmd) {
	switch m.active {
	case tabDistros:
		d, ok := m.selectedDistro()
		if !ok {
			return m, nil
		}
		return m, tea.ExecProcess(wsl.ShellCommand(d.Name), shellFinished)

	case tabContainers:
		c, ok := m.selectedContainer()
		if !ok {
			return m, nil
		}
		if !c.IsRunning() {
			m.notice = c.Name() + " çalışmıyor; önce s ile başlat"
			m.noticeErr, m.noticeAt = true, time.Now()
			return m, nil
		}

		cmd, err := wslc.ShellCommand(c.Name())
		if err != nil {
			m.notice, m.noticeErr, m.noticeAt = err.Error(), true, time.Now()
			return m, nil
		}
		return m, tea.ExecProcess(cmd, shellFinished)
	}

	return m, nil
}

func shellFinished(err error) tea.Msg { return shellDoneMsg{err} }

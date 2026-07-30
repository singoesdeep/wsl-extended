package ui

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/singoesdeep/wsl-extended/internal/ui/theme"
	"github.com/singoesdeep/wsl-extended/internal/wslc"
)

const (
	// logTail, panel açılırken geçmişten kaç satır çekileceği.
	logTail = 200
	// logCapacity, bellekte tutulan en fazla satır sayısı. Uzun süre açık
	// kalan bir panelin belleği sınırsız büyütmesini engeller.
	logCapacity = 2000
)

type logLineMsg string

// logClosedMsg, akışın sona erdiğini bildirir (kapsayıcı durdu ya da panel
// kapatıldı).
type logClosedMsg struct{}

type logErrMsg struct{ err error }

// logModel, tek bir kapsayıcının günlüklerini canlı gösteren paneldir.
type logModel struct {
	active bool
	target string

	lines []string
	ch    <-chan string
	// cancel, akışı başlatan context'i iptal eder; panel kapanırken çağrılmazsa
	// arka planda wslc süreci çalışmaya devam eder.
	cancel context.CancelFunc

	// follow açıkken panel her yeni satırda sona kayar.
	follow bool
	// offset, gösterilen ilk satırın indeksi.
	offset int

	err error
}

// startLogs, akışı başlatır ve ilk satırı bekleyen komutu döndürür.
func startLogs(target string) (logModel, tea.Cmd) {
	ctx, cancel := context.WithCancel(context.Background())

	m := logModel{active: true, target: target, follow: true, cancel: cancel}

	ch, err := wslc.StreamLogs(ctx, target, logTail)
	if err != nil {
		cancel()
		m.err = err
		return m, nil
	}
	m.ch = ch

	return m, waitForLine(ch)
}

// waitForLine, kanaldan tek bir satır okur. Her satır alındığında yeniden
// çağrılır; Bubble Tea'de sürekli akışı modele bağlamanın yolu budur.
func waitForLine(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-ch
		if !ok {
			return logClosedMsg{}
		}
		return logLineMsg(line)
	}
}

func (m logModel) close() logModel {
	if m.cancel != nil {
		m.cancel()
	}
	return logModel{}
}

func (m logModel) append(line string) logModel {
	m.lines = append(m.lines, line)
	if len(m.lines) > logCapacity {
		m.lines = m.lines[len(m.lines)-logCapacity:]
		m.offset = max(0, m.offset-1)
	}
	return m
}

// visibleLines, panelin gövdesine sığan satırları döndürür.
func (m logModel) visibleLines(height int) []string {
	if len(m.lines) == 0 {
		return nil
	}

	start := m.offset
	if m.follow {
		start = max(0, len(m.lines)-height)
	}
	start = min(start, max(0, len(m.lines)-1))

	end := min(len(m.lines), start+height)
	return m.lines[start:end]
}

// update, panel açıkken tuşları işler. İkinci dönüş değeri true ise panel
// kapatılmalıdır.
func (m logModel) update(msg tea.KeyMsg, height int) (logModel, bool) {
	switch msg.String() {
	case "esc", "q", "L":
		return m, true

	case "down", "j":
		// Elle kaydırmak takibi bırakır; kullanıcı okurken ekran altına kaçmaz.
		m.follow = false
		m.offset = min(m.offset+1, max(0, len(m.lines)-1))

	case "up", "k":
		m.follow = false
		m.offset = max(0, m.offset-1)

	case "pgdown":
		m.follow = false
		m.offset = min(m.offset+height, max(0, len(m.lines)-1))

	case "pgup":
		m.follow = false
		m.offset = max(0, m.offset-height)

	case "g", "home":
		m.follow = false
		m.offset = 0

	case "G", "end", "f":
		m.follow = true
	}
	return m, false
}

func (m logModel) view(width, height int) string {
	if m.err != nil {
		return theme.Error.Render("Günlükler açılamadı: " + m.err.Error())
	}

	head := theme.DialogTitle.Render("Günlükler · " + m.target)
	if m.follow {
		head += theme.DialogHint.Render("  (takip ediliyor)")
	} else {
		head += theme.DialogHint.Render("  (duraklatıldı — G ile takibe dön)")
	}

	body := m.visibleLines(height - 1)
	if len(body) == 0 {
		return head + "\n" + theme.Empty.Render("Henüz günlük satırı yok.")
	}

	var b strings.Builder
	b.WriteString(head)
	for _, line := range body {
		b.WriteString("\n")
		b.WriteString(theme.Row.Render(fitCell(line, max(1, width-2))))
	}
	return b.String()
}

// Package ui, uygulamanın Bubble Tea katmanıdır.
//
// Faz 1 kapsamı salt okunurdur: listeler, gezinme ve yenileme. Durum
// değiştiren komutlar (start/stop/remove) Faz 2'de onay diyaloğuyla gelecek.
package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/singoesdeep/wsl-extended/internal/ui/theme"
	"github.com/singoesdeep/wsl-extended/internal/wsl"
	"github.com/singoesdeep/wsl-extended/internal/wslc"
)

// refreshInterval, yalnızca etkin sekme için geçerli otomatik yenileme aralığı.
const refreshInterval = 2 * time.Second

// commandTimeout, tek bir CLI çağrısının bekleyebileceği en uzun süre.
const commandTimeout = 15 * time.Second

type tabID int

const (
	tabDistros tabID = iota
	tabContainers
	tabImages
	tabVolumes
	tabNetworks
	tabCount
)

var tabNames = [tabCount]string{"Distros", "Containers", "Images", "Volumes", "Networks"}

// needsWSLC, sekmenin wslc.exe gerektirip gerektirmediğini söyler.
func (t tabID) needsWSLC() bool { return t != tabDistros }

type distrosMsg struct {
	items []wsl.Distro
	err   error
}

type containersMsg struct {
	items []wslc.Container
	err   error
}

type imagesMsg struct {
	items []wslc.Image
	err   error
}

type volumesMsg struct {
	items []wslc.Volume
	err   error
}

type networksMsg struct {
	items []wslc.Network
	err   error
}

type versionMsg string

type tickMsg time.Time

// Model, uygulamanın kök modelidir.
type Model struct {
	width, height int

	active  tabID
	cursors [tabCount]int
	errs    [tabCount]error

	distros    []wsl.Distro
	containers []wslc.Container
	images     []wslc.Image
	volumes    []wslc.Volume
	networks   []wslc.Network

	wslOK      bool
	wslcOK     bool
	wslVersion string

	lastRefresh time.Time
	showHelp    bool
}

// New, başlangıç modelini kurar.
func New() Model {
	return Model{
		wslOK:  wsl.Available(),
		wslcOK: wslc.Available(),
	}
}

func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{tick(), loadVersion()}
	for t := tabDistros; t < tabCount; t++ {
		if c := m.load(t); c != nil {
			cmds = append(cmds, c)
		}
	}
	return tea.Batch(cmds...)
}

func tick() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func loadVersion() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
		defer cancel()
		v, err := wsl.Version(ctx)
		if err != nil {
			return versionMsg("")
		}
		return versionMsg(v)
	}
}

// load, verilen sekmenin verisini arka planda çeken komutu döndürür. Gerekli
// araç yoksa nil döner ve sekme kendi bilgi ekranını gösterir.
func (m Model) load(t tabID) tea.Cmd {
	if t == tabDistros && !m.wslOK {
		return nil
	}
	if t.needsWSLC() && !m.wslcOK {
		return nil
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
		defer cancel()

		switch t {
		case tabDistros:
			items, err := wsl.List(ctx)
			return distrosMsg{items, err}
		case tabContainers:
			items, err := wslc.Containers(ctx)
			return containersMsg{items, err}
		case tabImages:
			items, err := wslc.Images(ctx)
			return imagesMsg{items, err}
		case tabVolumes:
			items, err := wslc.Volumes(ctx)
			return volumesMsg{items, err}
		case tabNetworks:
			items, err := wslc.Networks(ctx)
			return networksMsg{items, err}
		}
		return nil
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tickMsg:
		// Yalnızca görünen sekme yenilenir; arka plandaki sekmeler için
		// sürekli süreç doğurmanın anlamı yok.
		return m, tea.Batch(tick(), m.load(m.active))

	case versionMsg:
		m.wslVersion = string(msg)
		return m, nil

	case distrosMsg:
		m.distros, m.errs[tabDistros] = msg.items, msg.err
		m.lastRefresh = time.Now()
		m.clampCursor(tabDistros)
		return m, nil

	case containersMsg:
		m.containers, m.errs[tabContainers] = msg.items, msg.err
		m.lastRefresh = time.Now()
		m.clampCursor(tabContainers)
		return m, nil

	case imagesMsg:
		m.images, m.errs[tabImages] = msg.items, msg.err
		m.lastRefresh = time.Now()
		m.clampCursor(tabImages)
		return m, nil

	case volumesMsg:
		m.volumes, m.errs[tabVolumes] = msg.items, msg.err
		m.lastRefresh = time.Now()
		m.clampCursor(tabVolumes)
		return m, nil

	case networksMsg:
		m.networks, m.errs[tabNetworks] = msg.items, msg.err
		m.lastRefresh = time.Now()
		m.clampCursor(tabNetworks)
		return m, nil
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "?":
		m.showHelp = !m.showHelp
		return m, nil

	case "tab", "l", "right":
		m.active = (m.active + 1) % tabCount
		return m, m.load(m.active)

	case "shift+tab", "h", "left":
		m.active = (m.active - 1 + tabCount) % tabCount
		return m, m.load(m.active)

	case "1", "2", "3", "4", "5":
		m.active = tabID(msg.String()[0] - '1')
		return m, m.load(m.active)

	case "down", "j":
		if n := m.count(m.active); n > 0 {
			m.cursors[m.active] = min(m.cursors[m.active]+1, n-1)
		}
		return m, nil

	case "up", "k":
		m.cursors[m.active] = max(m.cursors[m.active]-1, 0)
		return m, nil

	case "g", "home":
		m.cursors[m.active] = 0
		return m, nil

	case "G", "end":
		m.cursors[m.active] = max(m.count(m.active)-1, 0)
		return m, nil

	case "r":
		return m, m.load(m.active)
	}

	return m, nil
}

func (m Model) count(t tabID) int {
	switch t {
	case tabDistros:
		return len(m.distros)
	case tabContainers:
		return len(m.containers)
	case tabImages:
		return len(m.images)
	case tabVolumes:
		return len(m.volumes)
	case tabNetworks:
		return len(m.networks)
	}
	return 0
}

// clampCursor, liste kısaldığında imlecin listenin dışında kalmasını önler.
func (m *Model) clampCursor(t tabID) {
	if n := m.count(t); m.cursors[t] >= n {
		m.cursors[t] = max(n-1, 0)
	}
}

func (m Model) View() string {
	if m.width == 0 {
		return "yükleniyor…"
	}

	header := m.viewHeader()
	status := m.viewStatus()
	help := m.viewHelp()

	chrome := lipgloss.Height(header) + lipgloss.Height(status) + lipgloss.Height(help)
	bodyHeight := max(3, m.height-chrome-1)

	body := m.viewBody(bodyHeight)

	// Gövdeyi sabit yükseklikte tutmak, sekme değişiminde durum çubuğunun
	// zıplamasını engeller.
	if pad := bodyHeight - lipgloss.Height(body); pad > 0 {
		body += strings.Repeat("\n", pad)
	}

	return strings.Join([]string{header, body, help, status}, "\n")
}

func (m Model) viewHeader() string {
	var tabs []string
	for t := tabDistros; t < tabCount; t++ {
		label := fmt.Sprintf("%d %s", int(t)+1, tabNames[t])
		if n := m.count(t); n > 0 {
			label += fmt.Sprintf(" (%d)", n)
		}
		if t == m.active {
			tabs = append(tabs, theme.TabActive.Render(label))
		} else {
			tabs = append(tabs, theme.TabInactive.Render(label))
		}
	}
	return theme.TabBar.Render(lipgloss.JoinHorizontal(lipgloss.Top, tabs...))
}

func (m Model) viewBody(height int) string {
	if m.active == tabDistros && !m.wslOK {
		return theme.Error.Render("wsl.exe bulunamadı. WSL kurulu mu?")
	}
	if m.active.needsWSLC() && !m.wslcOK {
		return theme.Error.Render(
			"wslc.exe bulunamadı — kapsayıcı özellikleri devre dışı.\n" +
				"WSL 2.9 veya üstü ile birlikte gelir.")
	}
	if err := m.errs[m.active]; err != nil {
		return theme.Error.Render("Hata: " + err.Error())
	}

	cols, rows := m.tableData()
	if len(rows) == 0 {
		return theme.Empty.Render(m.emptyMessage())
	}
	return renderTable(cols, rows, m.cursors[m.active], m.width, height)
}

func (m Model) tableData() ([]column, [][]string) {
	switch m.active {
	case tabDistros:
		cols := []column{
			{title: "NAME"},
			{title: "STATE", width: 12, state: true},
			{title: "VERSION", width: 7},
			{title: "DEFAULT", width: 7},
		}
		rows := make([][]string, 0, len(m.distros))
		for _, d := range m.distros {
			def := ""
			if d.Default {
				def = "✓"
			}
			rows = append(rows, []string{d.Name, string(d.State), d.Version, def})
		}
		return cols, rows

	case tabContainers:
		cols := []column{
			{title: "NAME"},
			{title: "IMAGE"},
			{title: "STATE", width: 12, state: true},
			{title: "STATUS", width: 20},
			{title: "PORTS", width: 20},
		}
		rows := make([][]string, 0, len(m.containers))
		for _, c := range m.containers {
			rows = append(rows, []string{
				c.Name(), c.Image.String(), c.State.String(),
				c.Status.String(), c.Ports.String(),
			})
		}
		return cols, rows

	case tabImages:
		cols := []column{
			{title: "REPOSITORY"},
			{title: "TAG", width: 16},
			{title: "ID", width: 14},
			{title: "SIZE", width: 10},
			{title: "CREATED", width: 20},
		}
		rows := make([][]string, 0, len(m.images))
		for _, i := range m.images {
			rows = append(rows, []string{
				i.Repository.String(), i.Tag.String(), i.ID.String(),
				i.Size.String(), i.CreatedAt.String(),
			})
		}
		return cols, rows

	case tabVolumes:
		cols := []column{
			{title: "NAME"},
			{title: "DRIVER", width: 12},
			{title: "MOUNTPOINT"},
		}
		rows := make([][]string, 0, len(m.volumes))
		for _, v := range m.volumes {
			rows = append(rows, []string{
				v.Name.String(), v.Driver.String(), v.Mountpoint.String(),
			})
		}
		return cols, rows

	case tabNetworks:
		cols := []column{
			{title: "NAME"},
			{title: "DRIVER", width: 14},
			{title: "SCOPE", width: 12},
			{title: "ID", width: 16},
		}
		rows := make([][]string, 0, len(m.networks))
		for _, n := range m.networks {
			rows = append(rows, []string{
				n.Name.String(), n.Driver.String(), n.Scope.String(), n.ID.String(),
			})
		}
		return cols, rows
	}
	return nil, nil
}

func (m Model) emptyMessage() string {
	switch m.active {
	case tabDistros:
		return "Kurulu distro yok.  wsl --install ile bir tane ekleyebilirsin."
	case tabContainers:
		return "Kapsayıcı yok.  wslc run ile bir tane başlatabilirsin."
	case tabImages:
		return "İmaj yok.  wslc pull ile bir imaj çekebilirsin."
	case tabVolumes:
		return "Birim yok.  wslc volume create ile oluşturabilirsin."
	case tabNetworks:
		return "Ağ yok.  wslc network create ile oluşturabilirsin."
	}
	return "Kayıt yok."
}

func (m Model) viewHelp() string {
	keys := [][2]string{
		{"tab/1-5", "sekme"},
		{"j/k", "gezin"},
		{"r", "yenile"},
		{"?", "yardım"},
		{"q", "çık"},
	}
	if m.showHelp {
		keys = append(keys,
			[2]string{"g/G", "başa/sona"},
			[2]string{"h/l", "sekme değiştir"},
		)
	}

	var parts []string
	for _, k := range keys {
		parts = append(parts, theme.HelpKey.Render(k[0])+" "+k[1])
	}
	return theme.Help.Render(strings.Join(parts, "  ·  "))
}

func (m Model) viewStatus() string {
	parts := []string{"wsl-extended"}
	if m.wslVersion != "" {
		parts = append(parts, "WSL "+m.wslVersion)
	}
	if !m.wslcOK {
		parts = append(parts, "wslc yok")
	}
	if !m.lastRefresh.IsZero() {
		parts = append(parts, "yenilendi "+m.lastRefresh.Format("15:04:05"))
	}
	parts = append(parts, "salt okunur (Faz 1)")

	return theme.StatusBar.Render(strings.Join(parts, "  ·  "))
}

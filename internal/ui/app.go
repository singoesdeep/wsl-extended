// Package ui, uygulamanın Bubble Tea katmanıdır.
//
// Faz 1 kapsamı salt okunurdur: listeler, gezinme ve yenileme. Durum
// değiştiren komutlar (start/stop/remove) Faz 2'de onay diyaloğuyla gelecek.
package ui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/singoesdeep/wsl-extended/internal/store"
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
	tabStore
	tabContainers
	tabImages
	tabVolumes
	tabNetworks
	tabCount
)

var tabNames = [tabCount]string{
	"Distros", "Store", "Containers", "Images", "Volumes", "Networks",
}

// needsWSLC, sekmenin wslc.exe gerektirip gerektirmediğini söyler. Distro ve
// mağaza sekmeleri yalnızca wsl.exe kullanır.
func (t tabID) needsWSLC() bool {
	return t != tabDistros && t != tabStore
}

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

	// filter, etkin sekmedeki listeyi süzer. filtering true iken tuşlar
	// filtre metnine yazılır.
	filter    string
	filtering bool

	// marked, toplu işlem için işaretli satırların gerçek indeksleri.
	marked map[tabID]map[int]bool

	distros    []wsl.Distro
	containers []wslc.Container
	images     []wslc.Image
	volumes    []wslc.Volume
	networks   []wslc.Network

	wslOK      bool
	wslcOK     bool
	wslVersion string

	// data, uygulamanın kendi kalıcı verisi (şu an distro takma adları).
	data     *store.Data
	dataPath string

	lastRefresh time.Time
	showHelp    bool

	// confirm açıkken tüm tuşlar diyaloga gider.
	confirm confirmModel

	online []wsl.OnlineDistro

	logs    logModel
	stats   statsModel
	prompt  promptModel
	config  configModel
	inspect inspectModel
	menu    menuModel
	system  systemModel

	// progressPath boş değilken, süren işin yazdığı dosya büyüdükçe ilerleme
	// gösterilir. wsl --export yüzde bildirmediği için ilerleme, dosyanın o
	// ana kadarki boyutundan okunur.
	progressPath  string
	progressBytes int64

	// busy, bir işlem sürerken otomatik yenilemeyi durdurur; aksi hâlde liste
	// kullanıcının altından kayar.
	busy      bool
	busyLabel string

	notice    string
	noticeErr bool
	noticeAt  time.Time
}

// noticeTTL, işlem sonucu bildiriminin ekranda kalma süresi.
const noticeTTL = 6 * time.Second

// New, başlangıç modelini kurar.
func New() Model {
	path := store.Path()
	// Veri okunamazsa uygulama yine açılır; takma adlar boş kalır.
	data, _ := store.Load(path)

	return Model{
		wslOK:    wsl.Available(),
		wslcOK:   wslc.Available(),
		data:     data,
		dataPath: path,
		marked:   map[tabID]map[int]bool{},
	}
}

// displayName, distronun listede görünecek adını verir.
func (m Model) displayName(real string) string {
	return m.data.Alias(real)
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
		case tabStore:
			items, err := wsl.ListOnline(ctx)
			return onlineMsg{items, err}
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
		if !m.noticeAt.IsZero() && time.Since(m.noticeAt) > noticeTTL {
			m.notice, m.noticeErr, m.noticeAt = "", false, time.Time{}
		}
		// İşlem sürerken ya da onay beklenirken liste tazelenmez: imlecin
		// altındaki satırın değişmesi yanlış hedefe işlem yapılmasına yol açar.
		if m.busy {
			// Süren işin yazdığı dosya büyüdükçe ilerleme güncellenir.
			if m.progressPath != "" {
				if fi, err := os.Stat(m.progressPath); err == nil {
					m.progressBytes = fi.Size()
				}
			}
			return m, tick()
		}
		if m.confirm.active || m.prompt.active || m.config.active {
			return m, tick()
		}
		// Günlük paneli kendi akışından beslenir; liste yenilemeye gerek yok.
		if m.logs.active {
			return m, tick()
		}
		if m.stats.active {
			return m, tea.Batch(tick(), loadStats())
		}
		return m, tea.Batch(tick(), m.load(m.active))

	case logLineMsg:
		// Panel kapandıysa geciken satırlar yok sayılır.
		if !m.logs.active {
			return m, nil
		}
		m.logs = m.logs.append(string(msg))
		return m, waitForLine(m.logs.ch)

	case logClosedMsg:
		if m.logs.active {
			m.logs = m.logs.append("— akış sona erdi —")
			m.logs.ch = nil
		}
		return m, nil

	case logErrMsg:
		m.logs.err = msg.err
		return m, nil

	case statsMsg:
		m.stats.items, m.stats.err = msg.items, msg.err
		return m, nil

	case onlineMsg:
		m.online, m.errs[tabStore] = msg.items, msg.err
		m.lastRefresh = time.Now()
		m.clampCursor(tabStore)
		return m, nil

	case actionAfterRefreshMsg:
		if msg.err == nil {
			if msg.distros != nil {
				m.distros = msg.distros
				m.clampCursor(tabDistros)
			}
			if msg.containers != nil {
				m.containers = msg.containers
				m.clampCursor(tabContainers)
			}
		}
		if act, ok := m.actionFor(msg.key); ok {
			m.confirm = newConfirm(m.applyBulk(act))
		}
		return m, nil

	case inspectMsg:
		m.busy, m.busyLabel = false, ""
		alias := ""
		if msg.err == nil {
			alias = m.displayName(msg.info.Name)
		}
		m.inspect = inspectModel{active: true, info: msg.info, alias: alias, err: msg.err}
		return m, nil

	case configLoadedMsg:
		m.busy, m.busyLabel = false, ""
		if msg.err != nil {
			m.notice, m.noticeErr, m.noticeAt = msg.err.Error(), true, time.Now()
			return m, nil
		}
		m.config = newConfigModel(msg)
		return m, nil

	case shellDoneMsg:
		if msg.err != nil {
			m.notice, m.noticeErr, m.noticeAt = msg.err.Error(), true, time.Now()
		}
		// Kabuktan dönerken ekran kirlenmiş olabilir; liste hemen tazelenir.
		return m, m.load(m.active)

	case installDoneMsg:
		m.noticeAt = time.Now()
		if msg.err != nil {
			m.notice, m.noticeErr = msg.name+" kurulamadı: "+msg.err.Error(), true
		} else {
			m.notice, m.noticeErr = msg.name+" kuruldu", false
		}
		// Distro listesi de değiştiği için iki sekme birden tazelenir.
		return m, tea.Batch(m.load(tabDistros), m.load(m.active))

	case pullDoneMsg:
		m.noticeAt = time.Now()
		if msg.err != nil {
			m.notice, m.noticeErr = msg.image+" çekilemedi: "+msg.err.Error(), true
		} else {
			m.notice, m.noticeErr = msg.image+" çekildi", false
		}
		return m, tea.Batch(m.load(tabImages), m.load(m.active))

	case systemMsg:
		m.busy, m.busyLabel = false, ""
		m.system = systemModel{
			active: true, status: msg.status, version: msg.version,
			total: msg.total, running: msg.running, diskTotal: msg.diskTotal,
			err: msg.err,
		}
		return m, nil

	case actionDoneMsg:
		m.busy, m.busyLabel = false, ""
		m.progressPath, m.progressBytes = "", 0
		m.noticeAt = time.Now()
		// İşlem sonrası liste değişeceği için işaretlerin indeksleri artık
		// güvenilir değil.
		m.clearMarks(m.active)
		if msg.err != nil {
			m.notice, m.noticeErr = msg.err.Error(), true
		} else {
			m.notice, m.noticeErr = msg.act.done, false
		}
		// İşlem sonrası liste hemen tazelenir ki sonuç görünsün.
		return m, m.load(m.active)

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
	// Onay diyaloğu açıkken hiçbir tuş arka plandaki listeye ulaşmaz.
	if m.confirm.active {
		act := m.confirm.act
		updated, confirmed := m.confirm.update(msg)
		m.confirm = updated
		if confirmed {
			m.notice, m.noticeErr = "", false

			// Kurulum ve imaj çekme terminali devralır: ilerleme çubuklarını
			// ancak gerçek konsola bağlıyken çizerler.
			switch act.kind {
			case actDistroInstall:
				return m, runInstall(act.target)
			case actImagePull:
				return m, runPull(act.target)
			}

			m.busy, m.busyLabel = true, act.title
			if act.kind == actDistroExport {
				m.progressPath, m.progressBytes = act.path, 0
			}
			return m, act.run()
		}
		return m, nil
	}

	// Filtre yazarken tuşlar metne gider; aksi hâlde harfler kısayol olarak
	// yorumlanır ve filtre yazılamaz.
	if m.filtering {
		switch msg.Type {
		case tea.KeyEsc:
			m.filter, m.filtering = "", false
		case tea.KeyEnter:
			m.filtering = false // filtre kalır, gezinmeye dönülür
		case tea.KeyBackspace:
			if m.filter != "" {
				r := []rune(m.filter)
				m.filter = string(r[:len(r)-1])
			}
		case tea.KeyRunes:
			m.filter += string(msg.Runes)
		case tea.KeySpace:
			m.filter += " "
		}
		m.clampCursor(m.active)
		return m, nil
	}

	if m.inspect.active {
		switch msg.String() {
		case "esc", "q", "v":
			m.inspect = inspectModel{}
		}
		return m, nil
	}

	if m.menu.active {
		kind := m.menu.kind
		updated, choice := m.menu.update(msg)
		m.menu = updated
		if choice != "" {
			return m.handleMenuChoice(kind, choice)
		}
		return m, nil
	}

	if m.system.active {
		switch msg.String() {
		case "esc", "q", "w":
			m.system = systemModel{}
		}
		return m, nil
	}

	// Yapılandırma editörü açıkken tuşlar editöre aittir.
	if m.config.active {
		updated, act, save := m.config.update(msg)
		m.config = updated
		if save {
			m.config = configModel{}
			m.confirm = newConfirm(act)
		}
		return m, nil
	}

	// Form açıkken tuşlar forma aittir.
	if m.prompt.active {
		updated, submitted := m.prompt.update(msg)
		m.prompt = updated
		if submitted {
			next, cmd := m.submitPrompt()
			return next, cmd
		}
		return m, nil
	}

	// Günlük paneli açıkken tuşlar panele aittir.
	if m.logs.active {
		updated, closed := m.logs.update(msg, m.bodyHeight())
		m.logs = updated
		if closed {
			m.logs = m.logs.close()
		}
		return m, nil
	}

	if m.stats.active {
		switch msg.String() {
		case "esc", "q", "t":
			m.stats = statsModel{}
		}
		return m, nil
	}

	// Bir işlem sürerken yeni işlem başlatılmaz; gezinme serbest kalır.
	if m.busy {
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k", "down", "j":
			// gezinmeye izin ver
		default:
			return m, nil
		}
	}

	// Başlat/durdur kararı listedeki duruma bakar; liste iki saniyede bir
	// yenilendiği için bayat olabilir ve WSL boştaki distroyu kendiliğinden
	// kapatabilir. Karar vermeden önce durum tazelenir.
	if msg.String() == "s" && (m.active == tabDistros || m.active == tabContainers) {
		return m, m.refreshThenAct("s")
	}

	if act, ok := m.actionFor(msg.String()); ok {
		m.confirm = newConfirm(m.applyBulk(act))
		return m, nil
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "/":
		m.filtering = true
		return m, nil

	case " ":
		m.toggleMark()
		return m, nil

	case "enter":
		// Mağazada enter kurulum başlatır, diğer sekmelerde kabuk açar.
		if m.active == tabStore {
			return m.confirmInstall()
		}
		return m.openShell()

	case "v":
		if m.active == tabDistros {
			if d, ok := m.selectedDistro(); ok {
				m.busy, m.busyLabel = true, "bilgiler toplanıyor"
				return m, loadInspect(d)
			}
		}
		return m, nil

	case "m":
		if m.active == tabDistros {
			if d, ok := m.selectedDistro(); ok {
				m.menu = diskMenu(m.displayName(d.Name), d.IsRunning())
			}
		}
		return m, nil

	case "o":
		if m.active == tabDistros {
			if d, ok := m.selectedDistro(); ok {
				m.menu = openMenu(m.displayName(d.Name))
			}
		}
		return m, nil

	case "w":
		m.busy, m.busyLabel = true, "sistem durumu okunuyor"
		return m, loadSystem()

	case "p":
		// İmaj çekme ve kapsayıcı çalıştırma kapsayıcı sekmelerine ait.
		if m.wslcOK && (m.active == tabContainers || m.active == tabImages) {
			m.menu = containerMenu()
		}
		return m, nil

	case "l":
		if m.active == tabContainers && m.wslcOK {
			c, ok := m.selectedContainer()
			if !ok {
				return m, nil
			}
			logs, cmd := startLogs(c.Name())
			m.logs = logs
			return m, cmd
		}
		return m, nil

	case "t":
		if m.wslcOK {
			m.stats = statsModel{active: true}
			return m, loadStats()
		}
		return m, nil

	case "e":
		if m.active == tabDistros {
			if d, ok := m.selectedDistro(); ok {
				m.prompt = newExportPrompt(d)
			}
		}
		return m, nil

	case "i":
		if m.active == tabDistros {
			m.prompt = newImportPrompt()
		}
		return m, nil

	case "n":
		// Görünen adı değiştirir; WSL'deki gerçek ad korunur.
		if m.active == tabDistros {
			if d, ok := m.selectedDistro(); ok {
				current := ""
				if m.data.HasAlias(d.Name) {
					current = m.data.Alias(d.Name)
				}
				m.prompt = newAliasPrompt(d.Name, current)
			}
		}
		return m, nil

	case "c":
		_, hasDistro := m.selectedDistro()
		m.menu = settingsMenu(hasDistro && m.active == tabDistros)
		return m, nil

	case "?":
		m.showHelp = !m.showHelp
		return m, nil

	case "tab", "right":
		m.active = (m.active + 1) % tabCount
		m.filter = "" // filtre sekmeye özeldir
		return m, m.load(m.active)

	case "shift+tab", "left":
		m.active = (m.active - 1 + tabCount) % tabCount
		m.filter = ""
		return m, m.load(m.active)

	case "1", "2", "3", "4", "5", "6":
		m.active = tabID(msg.String()[0] - '1')
		m.filter = ""
		return m, m.load(m.active)

	case "down", "j":
		if n := m.count(m.active); n > 0 {
			m.cursors[m.active] = min(m.cursors[m.active]+1, n-1)
		}
		return m, nil

	case "up", "k":
		m.cursors[m.active] = max(m.cursors[m.active]-1, 0)
		return m, nil

	case "home":
		m.cursors[m.active] = 0
		return m, nil

	case "end":
		m.cursors[m.active] = max(m.count(m.active)-1, 0)
		return m, nil

	case "esc":
		// Filtreyi ve işaretleri tek tuşla temizler.
		m.filter = ""
		m.clearMarks(m.active)
		return m, nil

	case "r":
		return m, m.load(m.active)
	}

	return m, nil
}

// count, sekmede görünen (filtre uygulanmış) satır sayısıdır.
func (m Model) count(t tabID) int { return len(m.visible(t)) }

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

	bar := theme.TabBar.Render(lipgloss.JoinHorizontal(lipgloss.Top, tabs...))
	// Sekme şeridini içerikten ayıran ince çizgi, çerçeve kullanmadan
	// bölümleri ayırır.
	return bar + "\n" + theme.Rule.Render(strings.Repeat("─", max(1, m.width)))
}

// bodyHeight, gövdeye ayrılan yaklaşık satır sayısı. Sayfa kaydırma miktarı
// gibi tuş işlemlerinde kullanılır; kesin çizim hesabı View içindedir.
func (m Model) bodyHeight() int { return max(3, m.height-6) }

func (m Model) viewBody(height int) string {
	// Onay diyaloğu listenin yerine geçer: arka planda hedefin değiştiğini
	// düşündürecek bir görüntü kalmaz.
	if m.confirm.active {
		return m.confirm.view(m.width)
	}
	if m.prompt.active {
		return m.prompt.view(m.width)
	}
	if m.config.active {
		return m.config.view(m.width, height)
	}
	if m.inspect.active {
		return m.inspect.view(m.width, height)
	}
	if m.menu.active {
		return m.menu.view(m.width)
	}
	if m.system.active {
		return m.system.view(m.width, height)
	}
	if m.logs.active {
		return m.logs.view(m.width, height)
	}
	if m.stats.active {
		return m.stats.view(m.width, height)
	}
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

	cols, rows, marks := m.tableData()
	if len(rows) == 0 {
		if m.filter != "" {
			return theme.Empty.Render("“" + m.filter + "” ile eşleşen kayıt yok.")
		}
		return theme.Empty.Render(m.emptyMessage())
	}
	return renderTable(cols, rows, m.cursors[m.active], m.width, height, marks)
}

// tableData, etkin sekmenin sütunlarını, görünen satırlarını ve her satırın
// işaretli olup olmadığını verir.
func (m Model) tableData() ([]column, [][]string, []bool) {
	vis := m.visible(m.active)

	marks := make([]bool, 0, len(vis))
	for _, i := range vis {
		marks = append(marks, m.isMarked(m.active, i))
	}

	switch m.active {
	case tabDistros:
		cols := []column{
			{title: "NAME"},
			{title: "STATE", width: 12, state: true},
			{title: "VERSION", width: 7},
			{title: "DEFAULT", width: 7},
		}
		rows := make([][]string, 0, len(vis))
		for _, i := range vis {
			d := m.distros[i]
			def := ""
			if d.Default {
				def = "✓"
			}
			rows = append(rows, []string{
				theme.StateDot(d.IsRunning()) + " " + m.displayName(d.Name),
				string(d.State), d.Version, def,
			})
		}
		return cols, rows, marks

	case tabStore:
		cols := []column{
			{title: "DISTRO"},
			{title: "FRIENDLY NAME"},
			{title: "DURUM", width: 12},
		}
		installed := m.installedNames()
		rows := make([][]string, 0, len(vis))
		for _, i := range vis {
			o := m.online[i]
			state := ""
			if installed[strings.ToLower(o.Name)] {
				state = "kurulu"
			}
			rows = append(rows, []string{o.Name, o.Friendly, state})
		}
		return cols, rows, marks

	case tabContainers:
		cols := []column{
			{title: "NAME"},
			{title: "IMAGE"},
			{title: "STATE", width: 12, state: true},
			{title: "STATUS", width: 20},
			{title: "PORTS", width: 20},
		}
		rows := make([][]string, 0, len(vis))
		for _, i := range vis {
			c := m.containers[i]
			rows = append(rows, []string{
				c.Name(), c.Image.String(), c.State.String(),
				c.Status.String(), c.Ports.String(),
			})
		}
		return cols, rows, marks

	case tabImages:
		cols := []column{
			{title: "REPOSITORY"},
			{title: "TAG", width: 16},
			{title: "ID", width: 14},
			{title: "SIZE", width: 10},
			{title: "CREATED", width: 20},
		}
		rows := make([][]string, 0, len(vis))
		for _, idx := range vis {
			i := m.images[idx]
			rows = append(rows, []string{
				i.Repository.String(), i.Tag.String(), i.ID.String(),
				i.Size.String(), i.CreatedAt.String(),
			})
		}
		return cols, rows, marks

	case tabVolumes:
		cols := []column{
			{title: "NAME"},
			{title: "DRIVER", width: 12},
			{title: "MOUNTPOINT"},
		}
		rows := make([][]string, 0, len(vis))
		for _, i := range vis {
			v := m.volumes[i]
			rows = append(rows, []string{
				v.Name.String(), v.Driver.String(), v.Mountpoint.String(),
			})
		}
		return cols, rows, marks

	case tabNetworks:
		cols := []column{
			{title: "NAME"},
			{title: "DRIVER", width: 14},
			{title: "SCOPE", width: 12},
			{title: "ID", width: 16},
		}
		rows := make([][]string, 0, len(vis))
		for _, i := range vis {
			n := m.networks[i]
			rows = append(rows, []string{
				n.Name.String(), n.Driver.String(), n.Scope.String(), n.ID.String(),
			})
		}
		return cols, rows, marks
	}
	return nil, nil, nil
}

func (m Model) emptyMessage() string {
	switch m.active {
	case tabDistros:
		return "Kurulu distro yok.  Store sekmesinden (2) bir tane kurabilirsin."
	case tabStore:
		return "Katalog boş.  İnternet bağlantısını kontrol et."
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
	if m.confirm.active {
		if m.confirm.requiresTyping() {
			return theme.Help.Render(
				theme.HelpKey.Render("adı yaz") + " onayla  ·  " +
					theme.HelpKey.Render("esc") + " iptal")
		}
		return theme.Help.Render(
			theme.HelpKey.Render("y") + " onayla  ·  " +
				theme.HelpKey.Render("n/esc") + " iptal")
	}

	if m.logs.active {
		return theme.Help.Render(
			theme.HelpKey.Render("j/k") + " kaydır  ·  " +
				theme.HelpKey.Render("G") + " takibe dön  ·  " +
				theme.HelpKey.Render("esc") + " kapat")
	}
	if m.stats.active || m.inspect.active {
		return theme.Help.Render(theme.HelpKey.Render("esc") + " kapat")
	}
	if m.menu.active {
		return theme.Help.Render(
			theme.HelpKey.Render("j/k") + " seç  ·  " +
				theme.HelpKey.Render("enter") + " uygula  ·  " +
				theme.HelpKey.Render("esc") + " kapat")
	}
	if m.active == tabStore {
		return theme.Help.Render(
			theme.HelpKey.Render("enter") + " kur  ·  " +
				theme.HelpKey.Render("j/k") + " gezin  ·  " +
				theme.HelpKey.Render("tab") + " sekme  ·  " +
				theme.HelpKey.Render("q") + " çık")
	}
	if m.config.active {
		if m.config.editing {
			return theme.Help.Render(
				theme.HelpKey.Render("enter") + " bitir  ·  " +
					theme.HelpKey.Render("esc") + " bitir")
		}
		return theme.Help.Render(
			theme.HelpKey.Render("j/k") + " alan  ·  " +
				theme.HelpKey.Render("enter") + " düzenle  ·  " +
				theme.HelpKey.Render("backspace") + " temizle  ·  " +
				theme.HelpKey.Render("s") + " kaydet  ·  " +
				theme.HelpKey.Render("esc") + " kapat")
	}
	if m.prompt.active {
		return theme.Help.Render(
			theme.HelpKey.Render("tab") + " alan  ·  " +
				theme.HelpKey.Render("enter") + " onayla  ·  " +
				theme.HelpKey.Render("esc") + " iptal")
	}

	// Tuşlar bağlama göre değişir: her sekmede yalnızca orada işe yarayanlar
	// gösterilir, böylece satır kısa kalır.
	keys := m.contextKeys()

	if m.showHelp {
		return m.viewHelpFull(keys)
	}

	const sep = "  ·  "
	sepWidth := lipgloss.Width(sep)

	items := make([]string, 0, len(keys))
	for _, k := range keys {
		items = append(items, theme.HelpKey.Render(k[0])+" "+k[1])
	}

	// theme.Help yatay dolgusu satırın iki yanına birer sütun ekler.
	budget := m.width - 2

	if all := strings.Join(items, sep); lipgloss.Width(all) <= budget {
		return theme.Help.Render(all)
	}

	// Sığmıyorsa, tam listeye yönlendiren eke yer ayrılır.
	tail := theme.HelpKey.Render("?") + " tümü"
	budget -= sepWidth + lipgloss.Width(tail)

	var parts []string
	used := 0
	for _, item := range items {
		cost := lipgloss.Width(item)
		if len(parts) > 0 {
			cost += sepWidth
		}
		if used+cost > budget {
			break
		}
		parts = append(parts, item)
		used += cost
	}

	return theme.Help.Render(strings.Join(append(parts, tail), sep))
}

// contextKeys, etkin sekmede geçerli olan tuşları önem sırasına göre verir.
func (m Model) contextKeys() [][2]string {
	nav := [][2]string{
		{"tab", "sekme"},
		{"j/k", "gezin"},
		{"/", "ara"},
	}

	var act [][2]string
	switch m.active {
	case tabDistros:
		act = [][2]string{
			{"s", "başlat/durdur"},
			{"enter", "kabuk"},
			{"v", "detay"},
			{"o", "aç"},
			{"e", "yedekle"},
			{"i", "içe aktar"},
			{"n", "yeniden adlandır"},
			{"m", "disk"},
			{"u", "varsayılan yap"},
			{"d", "sil"},
		}
	case tabStore:
		act = [][2]string{{"enter", "kur"}}
	case tabContainers:
		act = [][2]string{
			{"s", "başlat/durdur"},
			{"enter", "kabuk"},
			{"l", "günlük"},
			{"t", "kaynak"},
			{"p", "yeni"},
			{"x", "sonlandır"},
			{"d", "sil"},
		}
	case tabImages:
		act = [][2]string{{"p", "imaj çek"}, {"d", "sil"}}
	default:
		act = [][2]string{{"d", "sil"}}
	}

	common := [][2]string{
		{"space", "işaretle"},
		{"c", "ayarlar"},
		{"w", "sistem"},
		{"r", "yenile"},
		{"q", "çık"},
	}

	return append(append(nav, act...), common...)
}

// viewHelpFull, "?" ile açılan çok satırlı tam listeyi çizer.
func (m Model) viewHelpFull(keys [][2]string) string {
	const perRow = 4

	var b strings.Builder
	for i, k := range keys {
		if i > 0 && i%perRow == 0 {
			b.WriteString("\n")
		} else if i > 0 {
			b.WriteString("  ·  ")
		}
		b.WriteString(theme.HelpKey.Render(k[0]) + " " + k[1])
	}
	b.WriteString("\n" + theme.HelpKey.Render("1-6") + " sekmeye git  ·  " +
		theme.HelpKey.Render("home/end") + " başa/sona  ·  " +
		theme.HelpKey.Render("esc") + " filtreyi ve işaretleri temizle  ·  " +
		theme.HelpKey.Render("?") + " kapat")

	return theme.Help.Render(b.String())
}

func (m Model) viewStatus() string {
	// İşlem durumu ve sonucu, sabit bilgilerin önüne geçer.
	if m.busy {
		s := "⟳ " + m.busyLabel + "…"
		if m.progressBytes > 0 {
			s += "  " + humanBytes(m.progressBytes) + " yazıldı"
		}
		return theme.Busy.Render(s)
	}
	if m.notice != "" {
		if m.noticeErr {
			return theme.NoticeError.Render("✗ " + m.notice)
		}
		return theme.Notice.Render("✓ " + m.notice)
	}

	// Filtre yazılırken durum çubuğu giriş alanına dönüşür.
	if m.filtering {
		return theme.StatusBar.Render("/" + m.filter + "▏")
	}

	var parts []string
	if m.filter != "" {
		parts = append(parts, "süzgeç: "+m.filter+" ("+itoa(m.count(m.active))+")")
	}
	if n := m.markCount(m.active); n > 0 {
		parts = append(parts, itoa(n)+" işaretli")
	}
	if len(parts) == 0 {
		parts = append(parts, "wsl-extended")
	}

	// Takma ad kullanılıyorsa gerçek ad görünür kalmalı: komutlar onunla çalışır.
	if m.active == tabDistros {
		if d, ok := m.selectedDistro(); ok && m.data.HasAlias(d.Name) {
			parts = append(parts, m.data.Alias(d.Name)+" → "+d.Name)
		}
	}

	if m.wslVersion != "" {
		parts = append(parts, "WSL "+m.wslVersion)
	}
	if !m.wslcOK {
		parts = append(parts, "wslc yok")
	}
	if !m.lastRefresh.IsZero() {
		parts = append(parts, "yenilendi "+m.lastRefresh.Format("15:04:05"))
	}

	return theme.StatusBar.Render(strings.Join(parts, "  ·  "))
}

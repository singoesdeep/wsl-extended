package ui

import (
	"context"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/singoesdeep/wsl-extended/internal/ui/theme"
	"github.com/singoesdeep/wsl-extended/internal/wsl"
	"github.com/singoesdeep/wsl-extended/internal/wslconf"
)

type configTarget int

const (
	// configGlobal, Windows tarafındaki %UserProfile%\.wslconfig dosyasıdır.
	configGlobal configTarget = iota
	// configDistro, distro içindeki /etc/wsl.conf dosyasıdır.
	configDistro
)

// configModel, yapılandırma dosyalarını alan alan düzenleyen panosudur.
//
// Düzenleme dosyanın kendisinde değil, form değerlerinde yapılır; kaydetmeden
// önce fark gösterilir ve onay alınır.
type configModel struct {
	active bool
	target configTarget

	// subject, .wslconfig için dosya yolu, wsl.conf için distro adıdır.
	subject string

	// original, dosyanın diskteki hâli. Fark bunun üzerinden hesaplanır.
	original string

	fields []wslconf.Field
	values []string
	idx    int

	// editing true iken tuşlar seçili alanın değerine yazılır.
	editing bool
	err     string
}

type configLoadedMsg struct {
	target  configTarget
	subject string
	content string
	err     error
}

// loadGlobalConfig, .wslconfig dosyasını okur.
func loadGlobalConfig() tea.Cmd {
	return func() tea.Msg {
		path := wslconf.WSLConfigPath()
		f, err := wslconf.LoadFile(path)
		if err != nil {
			return configLoadedMsg{target: configGlobal, subject: path, err: err}
		}
		return configLoadedMsg{target: configGlobal, subject: path, content: f.String()}
	}
}

// loadDistroConfig, distro içindeki wsl.conf dosyasını okur. Komut distroyu
// başlatır.
func loadDistroConfig(name string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), actionTimeout)
		defer cancel()

		content, err := wsl.ReadConf(ctx, name)
		return configLoadedMsg{target: configDistro, subject: name, content: content, err: err}
	}
}

func newConfigModel(msg configLoadedMsg) configModel {
	fields := wslconf.WSLConfigFields()
	if msg.target == configDistro {
		fields = wslconf.DistroConfFields()
	}

	file := wslconf.Parse(msg.content)
	return configModel{
		active:   true,
		target:   msg.target,
		subject:  msg.subject,
		original: msg.content,
		fields:   fields,
		values:   wslconf.Values(file, fields),
	}
}

func (c configModel) title() string {
	if c.target == configDistro {
		return "wsl.conf · " + c.subject
	}
	return ".wslconfig"
}

// update, panel açıkken tuşları işler. Dönen komut nil değilse kaydetme
// onayına geçilmiştir.
func (c configModel) update(msg tea.KeyMsg) (configModel, action, bool) {
	if c.editing {
		switch msg.Type {
		case tea.KeyEnter, tea.KeyEsc:
			c.editing = false
		case tea.KeyBackspace:
			if v := c.values[c.idx]; v != "" {
				r := []rune(v)
				c.values[c.idx] = string(r[:len(r)-1])
			}
			c.err = ""
		case tea.KeyRunes:
			c.values[c.idx] += string(msg.Runes)
			c.err = ""
		case tea.KeySpace:
			c.values[c.idx] += " "
			c.err = ""
		}
		return c, action{}, false
	}

	switch msg.String() {
	case "esc", "q":
		return configModel{}, action{}, false

	case "down", "j":
		c.idx = min(c.idx+1, len(c.fields)-1)

	case "up", "k":
		c.idx = max(c.idx-1, 0)

	case "enter":
		c.editing = true

	case "backspace", "delete":
		// Alanı boşaltmak, anahtarı dosyadan silmek demektir.
		c.values[c.idx] = ""

	case "s":
		return c.save()
	}

	return c, action{}, false
}

// save, değerleri doğrular ve kaydetme işini üretir.
func (c configModel) save() (configModel, action, bool) {
	for i, f := range c.fields {
		if err := f.Check(c.values[i]); err != nil {
			c.idx = i
			c.err = f.Label + ": " + err.Error()
			return c, action{}, false
		}
	}

	file := wslconf.Parse(c.original)
	wslconf.Apply(file, c.fields, c.values)
	updated := file.String()

	diff := wslconf.Diff(wslconf.Parse(c.original).Lines(), file.Lines())
	if !wslconf.HasChanges(diff) {
		c.err = "Değişiklik yok."
		return c, action{}, false
	}

	body := "Şu değişiklikler yazılacak:\n\n" + wslconf.FormatDiff(diff, 1)

	act := action{
		title:   "Yapılandırmayı kaydet · " + c.title(),
		body:    body,
		path:    c.subject,
		content: updated,
	}

	if c.target == configDistro {
		act.kind = actWriteDistroConf
		act.target = c.subject
		act.body += "\n\nÖnceki hâli /etc/wsl.conf.bak olarak saklanacak."
		act.done = c.subject + " için wsl.conf kaydedildi — etkili olması için distroyu yeniden başlat"
	} else {
		act.kind = actWriteGlobalConf
		// .wslconfig yalnızca WSL sanal makinesi yeniden kurulduğunda okunur;
		// bunu söylemezsek kullanıcı ayarın işe yaramadığını sanır.
		act.body += "\n\nÖnceki hâli .wslconfig.bak olarak saklanacak."
		act.done = ".wslconfig kaydedildi — etkili olması için X ile WSL'i kapat"
	}

	return c, act, true
}

func (c configModel) view(width, height int) string {
	var b strings.Builder

	b.WriteString(theme.DialogTitle.Render(c.title()))
	b.WriteString(theme.DialogHint.Render("   " + c.subject))
	b.WriteString("\n")

	labelWidth := 0
	for _, f := range c.fields {
		labelWidth = max(labelWidth, lipgloss.Width(f.Label))
	}
	valueWidth := max(12, min(width-labelWidth-10, 48))

	for i, f := range c.fields {
		selected := i == c.idx

		value := c.values[i]
		style := theme.Row
		if value == "" {
			value = "(ayarlanmamış)"
			style = theme.Help
		}
		if selected && c.editing {
			value = c.values[i] + "▏"
			style = theme.RowSelected
		} else if selected {
			style = theme.RowSelected
		}

		line := fitCell(f.Label, labelWidth) + "  " + fitCell(value, valueWidth)
		b.WriteString("\n")
		b.WriteString(style.Render(line))
	}

	b.WriteString("\n\n")
	if c.err != "" {
		b.WriteString(theme.NoticeError.Render(c.err))
	} else {
		b.WriteString(theme.DialogHint.Render("  " + c.fields[c.idx].Hint))
	}

	return b.String()
}

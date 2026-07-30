package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/singoesdeep/wsl-extended/internal/ui/theme"
)

// confirmModel, bir işlemi çalıştırmadan önce onay alan diyalogdur.
//
// İki kipi vardır: sıradan işlemler için y/n, geri dönüşü olmayanlar için
// hedefin adını harfi harfine yazdırma. İkincisi, yanlış satırdayken tek tuşa
// basıp veri kaybetmeyi imkânsız kılar.
type confirmModel struct {
	active bool
	act    action
	input  string
}

func newConfirm(a action) confirmModel {
	return confirmModel{active: true, act: a}
}

// requiresTyping, diyalogun ad yazdırma kipinde olup olmadığını söyler.
func (c confirmModel) requiresTyping() bool { return c.act.confirmWord != "" }

// satisfied, işlemin çalıştırılabilir olup olmadığını söyler.
func (c confirmModel) satisfied() bool {
	return !c.requiresTyping() || c.input == c.act.confirmWord
}

// update, diyalog açıkken tüm tuşları işler. İkinci dönüş değeri true ise
// kullanıcı onaylamıştır ve iş çalıştırılmalıdır.
func (c confirmModel) update(msg tea.KeyMsg) (confirmModel, bool) {
	switch msg.Type {
	case tea.KeyEsc:
		return confirmModel{}, false

	case tea.KeyEnter:
		if c.satisfied() {
			return confirmModel{}, true
		}
		return c, false

	case tea.KeyBackspace:
		if c.requiresTyping() && c.input != "" {
			r := []rune(c.input)
			c.input = string(r[:len(r)-1])
		}
		return c, false

	case tea.KeyRunes:
		if c.requiresTyping() {
			c.input += string(msg.Runes)
			return c, false
		}
		// y/n kipi: tek tuşla karar.
		switch strings.ToLower(string(msg.Runes)) {
		case "y", "e": // yes / evet
			return confirmModel{}, true
		case "n", "h": // no / hayır
			return confirmModel{}, false
		}
	}

	// Ctrl+C gibi tuşlar diyalogu kapatır; kazara işlem çalışmasın diye
	// onaylamış sayılmaz.
	if msg.Type == tea.KeyCtrlC {
		return confirmModel{}, false
	}
	return c, false
}

func (c confirmModel) view(width int) string {
	title := theme.DialogTitle
	if c.requiresTyping() {
		title = theme.DialogDangerTitle
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(title.Render(c.act.title))
	b.WriteString("\n")
	b.WriteString(rule(width))
	b.WriteString("\n\n")
	b.WriteString(theme.DialogBody.Render(c.act.body))
	b.WriteString("\n\n")

	if c.requiresTyping() {
		b.WriteString(theme.DialogHint.Render("Onaylamak için adı yaz: ") +
			theme.DialogWord.Render(c.act.confirmWord))
		b.WriteString("\n\n")

		field := c.input
		if field == "" {
			field = strings.Repeat(" ", lipgloss.Width(c.act.confirmWord))
		}
		style := theme.DialogInput
		if c.input != "" && !c.satisfied() {
			style = theme.DialogInputBad
		}
		if c.satisfied() {
			style = theme.DialogInputOK
		}
		b.WriteString("  " + style.Render(field))
		b.WriteString("\n\n")

		hint := "enter onayla  ·  esc iptal"
		if !c.satisfied() {
			hint = "adın tamamını yazana kadar enter çalışmaz  ·  esc iptal"
		}
		b.WriteString(theme.DialogHint.Render(hint))
	} else {
		b.WriteString(theme.DialogHint.Render("y onayla  ·  n iptal  ·  esc iptal"))
	}

	return b.String()
}

// rule, panelleri başlıklarından ayıran ince çizgi.
func rule(width int) string {
	return theme.Rule.Render(" " + strings.Repeat("─", max(4, min(width-2, 72))))
}

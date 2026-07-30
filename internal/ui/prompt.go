package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/singoesdeep/wsl-extended/internal/ui/theme"
)

type promptKind int

const (
	promptExport promptKind = iota
	promptImport
)

type promptField struct {
	label string
	value string
	hint  string
}

// promptModel, dosya yolu gibi serbest metin isteyen çok alanlı basit formdur.
type promptModel struct {
	active bool
	kind   promptKind
	title  string
	body   string
	fields []promptField
	idx    int
	err    string

	// subject, formun hangi distro için açıldığını taşır.
	subject string
}

// update, form açıkken tuşları işler. İkinci dönüş değeri true ise kullanıcı
// formu göndermiştir.
func (p promptModel) update(msg tea.KeyMsg) (promptModel, bool) {
	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		return promptModel{}, false

	case tea.KeyTab, tea.KeyDown:
		p.idx = (p.idx + 1) % len(p.fields)
		return p, false

	case tea.KeyShiftTab, tea.KeyUp:
		p.idx = (p.idx - 1 + len(p.fields)) % len(p.fields)
		return p, false

	case tea.KeyEnter:
		// Son alanda enter gönderir; öncekilerde sıradaki alana geçer.
		if p.idx < len(p.fields)-1 {
			p.idx++
			return p, false
		}
		return p, true

	case tea.KeyBackspace:
		v := p.fields[p.idx].value
		if v != "" {
			r := []rune(v)
			p.fields[p.idx].value = string(r[:len(r)-1])
		}
		p.err = ""
		return p, false

	case tea.KeyRunes, tea.KeySpace:
		if msg.Type == tea.KeySpace {
			p.fields[p.idx].value += " "
		} else {
			p.fields[p.idx].value += string(msg.Runes)
		}
		p.err = ""
		return p, false
	}

	return p, false
}

func (p promptModel) values() []string {
	out := make([]string, len(p.fields))
	for i, f := range p.fields {
		out[i] = strings.TrimSpace(f.value)
	}
	return out
}

func (p promptModel) view(width int) string {
	boxWidth := min(max(width-8, 40), 84)

	var b strings.Builder
	b.WriteString(theme.DialogTitle.Render(p.title))
	if p.body != "" {
		b.WriteString("\n\n")
		b.WriteString(p.body)
	}
	b.WriteString("\n")

	for i, f := range p.fields {
		b.WriteString("\n")
		b.WriteString(theme.DialogHint.Render(f.label))
		b.WriteString("\n")

		value := f.value
		style := theme.DialogInput
		if i == p.idx {
			style = theme.DialogInputOK
			value += "▏" // imleç
		}
		if value == "" {
			value = f.hint
			style = theme.DialogHint
		}
		b.WriteString(style.Render(fitCell(value, boxWidth-8)))
		b.WriteString("\n")
	}

	if p.err != "" {
		b.WriteString("\n")
		b.WriteString(theme.NoticeError.Render(p.err))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(theme.DialogHint.Render("tab alan değiştir  ·  enter onayla  ·  esc iptal"))

	return lipgloss.NewStyle().Padding(1, 2).Render(
		theme.Dialog.Width(boxWidth).Render(b.String()))
}

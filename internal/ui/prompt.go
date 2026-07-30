package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/singoesdeep/wsl-extended/internal/ui/theme"
)

type promptKind int

const (
	promptExport promptKind = iota
	promptImport
	promptAlias
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
	fieldWidth := min(max(width-12, 24), 72)

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(theme.DialogTitle.Render(p.title))
	b.WriteString("\n")
	b.WriteString(rule(width))
	if p.body != "" {
		b.WriteString("\n\n")
		b.WriteString(theme.DialogBody.Render(p.body))
	}
	b.WriteString("\n")

	for i, f := range p.fields {
		selected := i == p.idx

		mark := "  "
		if selected {
			mark = theme.Mark.Render(theme.SelectionMark) + " "
		}

		b.WriteString("\n")
		b.WriteString(" " + mark + theme.Label.Render(f.label))
		b.WriteString("\n")

		value, style := f.value, theme.DialogInput
		if selected {
			value += "▏" // imleç
			style = theme.DialogInputOK
		}
		if f.value == "" && !selected {
			value, style = f.hint, theme.DialogHint
		}
		b.WriteString("    " + style.Render(fitCell(value, fieldWidth)))
		b.WriteString("\n")
	}

	if p.err != "" {
		b.WriteString("\n")
		b.WriteString(theme.NoticeError.Render(p.err))
		b.WriteString("\n")
	}

	return b.String()
}

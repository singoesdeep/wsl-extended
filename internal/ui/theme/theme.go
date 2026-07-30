// Package theme, arayüzün tüm renk ve stil tanımlarını tek yerde toplar.
package theme

import "github.com/charmbracelet/lipgloss"

var (
	Accent  = lipgloss.AdaptiveColor{Light: "#0B6BCB", Dark: "#7AA2F7"}
	Subtle  = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#6C7086"}
	Text    = lipgloss.AdaptiveColor{Light: "#1F2328", Dark: "#C0CAF5"}
	Success = lipgloss.AdaptiveColor{Light: "#1A7F37", Dark: "#9ECE6A"}
	Warning = lipgloss.AdaptiveColor{Light: "#9A6700", Dark: "#E0AF68"}
	Danger  = lipgloss.AdaptiveColor{Light: "#B42318", Dark: "#F7768E"}
)

var (
	// TabActive / TabInactive, üstteki sekme şeridini oluşturur.
	TabActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(Accent).
			Padding(0, 2)

	TabInactive = lipgloss.NewStyle().
			Foreground(Subtle).
			Padding(0, 2)

	TabBar = lipgloss.NewStyle().Padding(0, 1)

	Header = lipgloss.NewStyle().
		Bold(true).
		Foreground(Subtle).
		Padding(0, 1)

	Row = lipgloss.NewStyle().Padding(0, 1)

	RowSelected = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(Accent).
			Padding(0, 1)

	StatusBar = lipgloss.NewStyle().
			Foreground(Subtle).
			Padding(0, 1)

	Empty = lipgloss.NewStyle().
		Foreground(Subtle).
		Padding(1, 2).
		Italic(true)

	Error = lipgloss.NewStyle().
		Foreground(Danger).
		Padding(1, 2)

	Help = lipgloss.NewStyle().
		Foreground(Subtle).
		Padding(0, 1)

	HelpKey = lipgloss.NewStyle().
		Bold(true).
		Foreground(Text)

	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(Accent)
)

// StateStyle, bir durum metnine (Running/Stopped/…) uygun rengi verir.
func StateStyle(state string) lipgloss.Style {
	switch state {
	case "Running", "running":
		return lipgloss.NewStyle().Foreground(Success)
	case "Stopped", "stopped", "exited", "Exited":
		return lipgloss.NewStyle().Foreground(Subtle)
	case "Unknown", "":
		return lipgloss.NewStyle().Foreground(Subtle)
	default:
		return lipgloss.NewStyle().Foreground(Warning)
	}
}

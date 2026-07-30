// Package theme, arayüzün tüm renk ve stil tanımlarını tek yerde toplar.
//
// Görsel dil minimal ve tipografiktir: çerçeve ve dolu arka plan kullanılmaz,
// hiyerarşi boşluk, hizalama, kalınlık ve renk tonuyla kurulur. Seçili satır
// dolu bir bloğa dönüşmediği için hücre bazlı renkler (durum renkleri gibi)
// seçiliyken de okunur kalır.
package theme

import "github.com/charmbracelet/lipgloss"

var (
	Accent  = lipgloss.AdaptiveColor{Light: "#0B6BCB", Dark: "#7AA2F7"}
	Text    = lipgloss.AdaptiveColor{Light: "#1F2328", Dark: "#C0CAF5"}
	Subtle  = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#7A819A"}
	Faint   = lipgloss.AdaptiveColor{Light: "#9CA3AF", Dark: "#565F89"}
	Success = lipgloss.AdaptiveColor{Light: "#1A7F37", Dark: "#9ECE6A"}
	Warning = lipgloss.AdaptiveColor{Light: "#9A6700", Dark: "#E0AF68"}
	Danger  = lipgloss.AdaptiveColor{Light: "#B42318", Dark: "#F7768E"}
)

// SelectionMark, seçili satırın solundaki ince işaretçi. Dolu arka plan yerine
// bu kullanılır: satırın kendi renkleri korunur.
const SelectionMark = "▍"

var (
	// Sekmeler: etkin olan kalın ve vurgulu, diğerleri soluk.
	TabActive = lipgloss.NewStyle().
			Bold(true).
			Foreground(Accent).
			Padding(0, 2)

	TabInactive = lipgloss.NewStyle().
			Foreground(Faint).
			Padding(0, 2)

	TabBar = lipgloss.NewStyle().Padding(0, 1)

	// Rule, bölümleri ayıran ince yatay çizgi.
	Rule = lipgloss.NewStyle().Foreground(Faint)

	// Header, sütun başlıkları: küçük ve soluk, içeriğin önüne geçmez.
	Header = lipgloss.NewStyle().
		Foreground(Faint).
		Padding(0, 1)

	Row = lipgloss.NewStyle().Padding(0, 1)

	RowSelected = lipgloss.NewStyle().Bold(true)

	// Mark, seçili satır işaretçisinin rengi.
	Mark = lipgloss.NewStyle().Foreground(Accent)

	Empty = lipgloss.NewStyle().
		Foreground(Subtle).
		Padding(1, 2)

	Error = lipgloss.NewStyle().
		Foreground(Danger).
		Padding(1, 2)

	StatusBar = lipgloss.NewStyle().
			Foreground(Faint).
			Padding(0, 1)

	Help = lipgloss.NewStyle().
		Foreground(Faint).
		Padding(0, 1)

	HelpKey = lipgloss.NewStyle().Foreground(Subtle)

	Title = lipgloss.NewStyle().Bold(true).Foreground(Text)

	// Label, detay panellerinde alan adı.
	Label = lipgloss.NewStyle().Foreground(Faint)

	// Value, detay panellerinde alan değeri.
	Value = lipgloss.NewStyle().Foreground(Text)
)

// Diyalog ve form stilleri. Çerçeve yerine başlık + ince ayırıcı kullanılır;
// tehlikeli işlemler renk ve metinle ayrışır, yalnızca renge güvenilmez.
var (
	DialogTitle = lipgloss.NewStyle().Bold(true).Foreground(Text).Padding(0, 1)

	DialogDangerTitle = lipgloss.NewStyle().Bold(true).Foreground(Danger).Padding(0, 1)

	DialogBody = lipgloss.NewStyle().Foreground(Text).Padding(0, 1)

	DialogHint = lipgloss.NewStyle().Foreground(Faint).Padding(0, 1)

	DialogWord = lipgloss.NewStyle().Bold(true).Foreground(Danger)

	// Giriş alanı: alt çizgi ile belirtilir, kutu çizilmez.
	DialogInput = lipgloss.NewStyle().Foreground(Text).Underline(true)

	DialogInputBad = lipgloss.NewStyle().Foreground(Danger).Underline(true)

	DialogInputOK = lipgloss.NewStyle().Foreground(Success).Underline(true)

	Notice = lipgloss.NewStyle().Foreground(Success).Padding(0, 1)

	NoticeError = lipgloss.NewStyle().Foreground(Danger).Padding(0, 1)

	Busy = lipgloss.NewStyle().Foreground(Warning).Padding(0, 1)

	// Fark görünümü.
	DiffAdd    = lipgloss.NewStyle().Foreground(Success)
	DiffRemove = lipgloss.NewStyle().Foreground(Danger)
	DiffSame   = lipgloss.NewStyle().Foreground(Faint)
)

// StateStyle, bir durum metnine uygun rengi verir.
func StateStyle(state string) lipgloss.Style {
	switch state {
	case "Running", "running":
		return lipgloss.NewStyle().Foreground(Success)
	case "Stopped", "stopped", "exited", "Exited":
		return lipgloss.NewStyle().Foreground(Faint)
	case "Unknown", "":
		return lipgloss.NewStyle().Foreground(Faint)
	default:
		return lipgloss.NewStyle().Foreground(Warning)
	}
}

// StateDot, durumu tek karakterlik bir göstergeye indirger. Metinle birlikte
// kullanılır; renk tek başına anlam taşımaz.
func StateDot(running bool) string {
	if running {
		return lipgloss.NewStyle().Foreground(Success).Render("●")
	}
	return lipgloss.NewStyle().Foreground(Faint).Render("○")
}

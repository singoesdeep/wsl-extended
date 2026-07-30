// Command wsl-extended, WSL distrolarını ve wslc kapsayıcılarını yöneten
// terminal arayüzünü başlatır.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/singoesdeep/wsl-extended/internal/ui"
)

func main() {
	p := tea.NewProgram(ui.New(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "wsl-extended:", err)
		os.Exit(1)
	}
}

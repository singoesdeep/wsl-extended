package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/singoesdeep/wsl-extended/internal/wslc"
)

type pullDoneMsg struct {
	image string
	err   error
}

// runPull, imaj indirmeyi terminali devrederek çalıştırır; indirme yüzdesi
// ancak gerçek konsolda görünür.
func runPull(image string) tea.Cmd {
	cmd, err := wslc.PullCommand(image)
	if err != nil {
		return func() tea.Msg { return pullDoneMsg{image: image, err: err} }
	}
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return pullDoneMsg{image: image, err: err}
	})
}

// submitPull, imaj adını doğrulayıp indirmeyi onaya sunar.
func (m Model) submitPull(image string) (Model, tea.Cmd) {
	if image == "" {
		m.prompt.err = "İmaj adı boş olamaz."
		return m, nil
	}

	m.prompt = promptModel{}
	m.confirm = newConfirm(action{
		kind: actImagePull, target: image, display: image,
		title: "İmaj çek",
		body: image + " indirilecek.\n" +
			"İndirme ekranı açılır; bitince arayüze dönersin.",
		done: image + " çekildi",
	})
	return m, nil
}

// submitRun, kapsayıcı çalıştırma formunu doğrulayıp onaya sunar.
func (m Model) submitRun(values []string) (Model, tea.Cmd) {
	opts := wslc.RunOptions{
		Image:   values[0],
		Name:    values[1],
		Port:    values[2],
		Volume:  values[3],
		Command: values[4],
	}
	if opts.Image == "" {
		m.prompt.err = "İmaj boş olamaz."
		return m, nil
	}

	// Kullanıcı çalıştırılacak komutu onaydan önce görebilmeli.
	preview := "wslc " + joinArgs(opts.RunArgs())

	m.prompt = promptModel{}
	m.confirm = newConfirm(action{
		kind: actContainerRun, display: opts.Image, runOpts: opts,
		title: "Kapsayıcı çalıştır",
		body:  "Şu komut çalıştırılacak:\n\n" + preview,
		done:  opts.Image + " başlatıldı",
	})
	return m, nil
}

func joinArgs(args []string) string {
	out := ""
	for i, a := range args {
		if i > 0 {
			out += " "
		}
		out += a
	}
	return out
}

package ui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/singoesdeep/wsl-extended/internal/wsl"
	"github.com/singoesdeep/wsl-extended/internal/wslc"
)

// actionTimeout, durum değiştiren komutlar için üst sınır. Listeleme
// komutlarından uzundur; distro durdurma ya da silme dakikalar sürebilir.
const actionTimeout = 3 * time.Minute

type actionKind int

const (
	actDistroStart actionKind = iota
	actDistroStop
	actDistroSetDefault
	actDistroUnregister
	actWSLShutdown
	actContainerStart
	actContainerStop
	actContainerKill
	actContainerRemove
	actImageRemove
	actVolumeRemove
	actNetworkRemove
)

// action, onaya sunulan ve onaylanınca çalıştırılacak iştir.
type action struct {
	kind    actionKind
	target  string // komuta geçirilecek ad ya da kimlik
	display string // kullanıcıya gösterilecek ad

	title string
	body  string

	// confirmWord boş değilse kullanıcı, işlemi onaylamak için bu metni harfi
	// harfine yazmak zorundadır. Geri dönüşü olmayan işlemler için kullanılır.
	confirmWord string

	// done, işlem başarıyla bittiğinde gösterilecek bildirim.
	done string
}

type actionDoneMsg struct {
	act action
	err error
}

func (a action) run() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), actionTimeout)
		defer cancel()

		var err error
		switch a.kind {
		case actDistroStart:
			err = wsl.Start(ctx, a.target)
		case actDistroStop:
			err = wsl.Terminate(ctx, a.target)
		case actDistroSetDefault:
			err = wsl.SetDefault(ctx, a.target)
		case actDistroUnregister:
			err = wsl.Unregister(ctx, a.target)
		case actWSLShutdown:
			err = wsl.Shutdown(ctx)
		case actContainerStart:
			err = wslc.StartContainer(ctx, a.target)
		case actContainerStop:
			err = wslc.StopContainer(ctx, a.target)
		case actContainerKill:
			err = wslc.KillContainer(ctx, a.target)
		case actContainerRemove:
			err = wslc.RemoveContainer(ctx, a.target)
		case actImageRemove:
			err = wslc.RemoveImage(ctx, a.target)
		case actVolumeRemove:
			err = wslc.RemoveVolume(ctx, a.target)
		case actNetworkRemove:
			err = wslc.RemoveNetwork(ctx, a.target)
		}

		return actionDoneMsg{act: a, err: err}
	}
}

// actionFor, etkin sekme ve seçili satır için basılan tuşa karşılık gelen işi
// üretir. İkinci dönüş değeri false ise tuşun bu bağlamda karşılığı yoktur.
func (m Model) actionFor(key string) (action, bool) {
	// Tüm WSL'i kapatmak seçili satırdan bağımsızdır.
	if key == "X" {
		return action{
			kind:  actWSLShutdown,
			title: "WSL'i kapat",
			body: "Tüm distrolar durdurulacak ve WSL sanal makinesi kapanacak.\n" +
				"Çalışan işler sonlanır. Veri kaybı yok.",
			done: "WSL kapatıldı",
		}, true
	}

	switch m.active {
	case tabDistros:
		d, ok := m.selectedDistro()
		if !ok {
			return action{}, false
		}
		switch key {
		case "s":
			return action{
				kind: actDistroStart, target: d.Name, display: d.Name,
				title: "Distroyu başlat",
				body:  d.Name + " başlatılacak.",
				done:  d.Name + " başlatıldı",
			}, true
		case "S":
			return action{
				kind: actDistroStop, target: d.Name, display: d.Name,
				title: "Distroyu durdur",
				body:  d.Name + " durdurulacak. İçinde çalışan işler sonlanır.",
				done:  d.Name + " durduruldu",
			}, true
		case "u":
			return action{
				kind: actDistroSetDefault, target: d.Name, display: d.Name,
				title: "Varsayılan yap",
				body:  d.Name + " varsayılan distro olacak.",
				done:  d.Name + " varsayılan yapıldı",
			}, true
		case "d":
			return action{
				kind: actDistroUnregister, target: d.Name, display: d.Name,
				title: "Distroyu kalıcı olarak sil",
				body: d.Name + " kaydından düşürülecek ve diskteki TÜM verisi silinecek.\n" +
					"Bu işlemin geri dönüşü yoktur. Yedeğin yoksa her şey kaybolur.",
				confirmWord: d.Name,
				done:        d.Name + " silindi",
			}, true
		}

	case tabContainers:
		c, ok := m.selectedContainer()
		if !ok {
			return action{}, false
		}
		name := c.Name()
		switch key {
		case "s":
			return action{
				kind: actContainerStart, target: name, display: name,
				title: "Kapsayıcıyı başlat",
				body:  name + " başlatılacak.",
				done:  name + " başlatıldı",
			}, true
		case "S":
			return action{
				kind: actContainerStop, target: name, display: name,
				title: "Kapsayıcıyı durdur",
				body:  name + " nazikçe durdurulacak.",
				done:  name + " durduruldu",
			}, true
		case "K":
			return action{
				kind: actContainerKill, target: name, display: name,
				title: "Kapsayıcıyı sonlandır",
				body:  name + " beklemeden sonlandırılacak. Kaydedilmemiş iş kaybolabilir.",
				done:  name + " sonlandırıldı",
			}, true
		case "d":
			return action{
				kind: actContainerRemove, target: name, display: name,
				title: "Kapsayıcıyı sil",
				body:  name + " silinecek. İmajı ve birimleri etkilenmez.",
				done:  name + " silindi",
			}, true
		}

	case tabImages:
		i, ok := m.selectedImage()
		if !ok {
			return action{}, false
		}
		ref := i.Repository.String()
		if t := i.Tag.String(); t != "" {
			ref += ":" + t
		}
		if ref == ":" || ref == "" {
			ref = i.ID.String()
		}
		if key == "d" {
			return action{
				kind: actImageRemove, target: ref, display: ref,
				title: "İmajı sil",
				body:  ref + " silinecek. Gerekirse yeniden çekilebilir.",
				done:  ref + " silindi",
			}, true
		}

	case tabVolumes:
		v, ok := m.selectedVolume()
		if !ok {
			return action{}, false
		}
		name := v.Name.String()
		if key == "d" {
			return action{
				kind: actVolumeRemove, target: name, display: name,
				title: "Birimi kalıcı olarak sil",
				body: name + " birimi ve içindeki TÜM veri silinecek.\n" +
					"Bu işlemin geri dönüşü yoktur.",
				confirmWord: name,
				done:        name + " silindi",
			}, true
		}

	case tabNetworks:
		n, ok := m.selectedNetwork()
		if !ok {
			return action{}, false
		}
		name := n.Name.String()
		if key == "d" {
			return action{
				kind: actNetworkRemove, target: name, display: name,
				title: "Ağı sil",
				body:  name + " ağı silinecek. Bağlı kapsayıcı varsa işlem başarısız olur.",
				done:  name + " silindi",
			}, true
		}
	}

	return action{}, false
}

func (m Model) selectedDistro() (wsl.Distro, bool) {
	i := m.cursors[tabDistros]
	if i < 0 || i >= len(m.distros) {
		return wsl.Distro{}, false
	}
	return m.distros[i], true
}

func (m Model) selectedContainer() (wslc.Container, bool) {
	i := m.cursors[tabContainers]
	if i < 0 || i >= len(m.containers) {
		return wslc.Container{}, false
	}
	return m.containers[i], true
}

func (m Model) selectedImage() (wslc.Image, bool) {
	i := m.cursors[tabImages]
	if i < 0 || i >= len(m.images) {
		return wslc.Image{}, false
	}
	return m.images[i], true
}

func (m Model) selectedVolume() (wslc.Volume, bool) {
	i := m.cursors[tabVolumes]
	if i < 0 || i >= len(m.volumes) {
		return wslc.Volume{}, false
	}
	return m.volumes[i], true
}

func (m Model) selectedNetwork() (wslc.Network, bool) {
	i := m.cursors[tabNetworks]
	if i < 0 || i >= len(m.networks) {
		return wslc.Network{}, false
	}
	return m.networks[i], true
}

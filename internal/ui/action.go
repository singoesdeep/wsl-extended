package ui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/singoesdeep/wsl-extended/internal/wsl"
	"github.com/singoesdeep/wsl-extended/internal/wslc"
	"github.com/singoesdeep/wsl-extended/internal/wslconf"
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
	actDistroExport
	actDistroImport
	actWriteGlobalConf
	actWriteDistroConf
	actDistroInstall
	actDistroResize
	actDistroSparse
	actDistroMove
	actWSLUpdate
	actImagePull
	actContainerRun
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

	// Yedekleme işlerine özgü alanlar.
	path       string // export hedefi, import kaynağı ya da yapılandırma yolu
	installDir string // import kurulum dizini
	stopFirst  bool   // export öncesi distroyu durdur
	content    string // yapılandırma dosyasının yazılacak hâli
	size       string // disk yeniden boyutlandırma değeri
	sparse     bool   // seyrek disk kipi

	runOpts wslc.RunOptions // yeni kapsayıcı seçenekleri

	// targets boş değilse iş, listedeki her hedef için sırayla çalıştırılır.
	targets []string
}

type actionDoneMsg struct {
	act action
	err error
}

func (a action) run() tea.Cmd {
	// Toplu iş: her hedef sırayla çalıştırılır, ilk hatada durulur. Kısmen
	// tamamlanmış bir toplu işlemi sessizce başarılı saymak yanıltıcı olurdu.
	if len(a.targets) > 0 {
		return func() tea.Msg {
			for _, t := range a.targets {
				single := a
				single.targets, single.target = nil, t

				msg := single.runOnce()
				if msg.err != nil {
					return actionDoneMsg{act: a, err: msg.err}
				}
			}
			return actionDoneMsg{act: a}
		}
	}

	return func() tea.Msg { return a.runOnce() }
}

func (a action) runOnce() actionDoneMsg {
	return func() actionDoneMsg {
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
		case actDistroExport:
			// Çalışan bir distronun dosya sistemi export sırasında değişebilir;
			// tutarlı bir arşiv için önce durdurulur.
			if a.stopFirst {
				if err = wsl.Terminate(ctx, a.target); err != nil {
					break
				}
			}
			err = wsl.Export(ctx, a.target, a.path)
		case actDistroImport:
			err = wsl.Import(ctx, a.target, a.installDir, a.path)
		case actWriteGlobalConf:
			err = wslconf.SaveFile(a.path, a.content)
		case actWriteDistroConf:
			err = wsl.WriteConf(ctx, a.target, a.content)
		case actDistroResize:
			err = wsl.Resize(ctx, a.target, a.size)
		case actDistroSparse:
			err = wsl.SetSparse(ctx, a.target, a.sparse)
		case actDistroMove:
			err = wsl.Move(ctx, a.target, a.path)
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
		case actWSLUpdate:
			err = wsl.Update(ctx)
		case actContainerRun:
			err = wslc.Run(ctx, a.runOpts)
		}

		return actionDoneMsg{act: a, err: err}
	}()
}

// actionFor, etkin sekme ve seçili satır için basılan tuşa karşılık gelen işi
// üretir. İkinci dönüş değeri false ise tuşun bu bağlamda karşılığı yoktur.
func (m Model) actionFor(key string) (action, bool) {
	switch m.active {
	case tabDistros:
		d, ok := m.selectedDistro()
		if !ok {
			return action{}, false
		}
		switch key {
		case "s":
			// Tek tuş duruma göre davranır: çalışanı durdurur, durmuşu başlatır.
			if d.IsRunning() {
				return action{
					kind: actDistroStop, target: d.Name, display: d.Name,
					title: "Distroyu durdur",
					body:  d.Name + " durdurulacak. İçinde çalışan işler sonlanır.",
					done:  d.Name + " durduruldu",
				}, true
			}
			return action{
				kind: actDistroStart, target: d.Name, display: d.Name,
				title: "Distroyu başlat",
				body:  d.Name + " başlatılacak.",
				done:  d.Name + " başlatıldı",
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
				// Yedek alma kısayolu bilerek buraya bağlanmadı: ad yazdırma
				// kipinde her tuş yazılan metnin parçasıdır, kısayol olamaz.
				body: d.Name + " kaydından düşürülecek ve diskteki TÜM verisi silinecek.\n" +
					"Bu işlemin geri dönüşü yoktur. Yedeğin yoksa her şey kaybolur.\n\n" +
					"Yedeğin yoksa esc ile çık ve önce e tuşuyla dışa aktar.",
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
			if c.IsRunning() {
				return action{
					kind: actContainerStop, target: name, display: name,
					title: "Kapsayıcıyı durdur",
					body:  name + " nazikçe durdurulacak.",
					done:  name + " durduruldu",
				}, true
			}
			return action{
				kind: actContainerStart, target: name, display: name,
				title: "Kapsayıcıyı başlat",
				body:  name + " başlatılacak.",
				done:  name + " başlatıldı",
			}, true
		case "x":
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

// Seçim yardımcıları filtreyi hesaba katar: imleç görünen satırlar arasında
// gezinir, hedef ise her zaman gerçek listedeki kayıttır.

func (m Model) selectedDistro() (wsl.Distro, bool) {
	i, ok := m.realIndex(tabDistros)
	if !ok || i >= len(m.distros) {
		return wsl.Distro{}, false
	}
	return m.distros[i], true
}

func (m Model) selectedContainer() (wslc.Container, bool) {
	i, ok := m.realIndex(tabContainers)
	if !ok || i >= len(m.containers) {
		return wslc.Container{}, false
	}
	return m.containers[i], true
}

func (m Model) selectedImage() (wslc.Image, bool) {
	i, ok := m.realIndex(tabImages)
	if !ok || i >= len(m.images) {
		return wslc.Image{}, false
	}
	return m.images[i], true
}

func (m Model) selectedVolume() (wslc.Volume, bool) {
	i, ok := m.realIndex(tabVolumes)
	if !ok || i >= len(m.volumes) {
		return wslc.Volume{}, false
	}
	return m.volumes[i], true
}

func (m Model) selectedNetwork() (wslc.Network, bool) {
	i, ok := m.realIndex(tabNetworks)
	if !ok || i >= len(m.networks) {
		return wslc.Network{}, false
	}
	return m.networks[i], true
}

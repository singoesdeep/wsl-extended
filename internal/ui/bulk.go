package ui

import (
	"fmt"
	"strings"
)

// bulkAllowed, toplu uygulanabilen işlemlerdir.
//
// Geri dönüşü olmayan iki işlem bilerek dışarıda: distro kaydını silme ve birim
// silme. Bunların koruması hedefin adını harfi harfine yazdırmaktır; toplu
// işlemde tek bir onay birden çok distroyu silerdi ve o koruma anlamını
// yitirirdi. Bu ikisi tek tek yapılır.
var bulkAllowed = map[actionKind]bool{
	actDistroStart:     true,
	actDistroStop:      true,
	actContainerStart:  true,
	actContainerStop:   true,
	actContainerKill:   true,
	actContainerRemove: true,
	actImageRemove:     true,
	actNetworkRemove:   true,
}

// targetName, sekmedeki gerçek indeksin komutlara geçirilecek adını verir.
func (m Model) targetName(t tabID, i int) string {
	switch t {
	case tabDistros:
		if i < len(m.distros) {
			return m.distros[i].Name
		}
	case tabContainers:
		if i < len(m.containers) {
			return m.containers[i].Name()
		}
	case tabImages:
		if i < len(m.images) {
			im := m.images[i]
			ref := im.Repository.String()
			if tag := im.Tag.String(); tag != "" {
				ref += ":" + tag
			}
			if ref == "" || ref == ":" {
				return im.ID.String()
			}
			return ref
		}
	case tabVolumes:
		if i < len(m.volumes) {
			return m.volumes[i].Name.String()
		}
	case tabNetworks:
		if i < len(m.networks) {
			return m.networks[i].Name.String()
		}
	}
	return ""
}

// displayFor, listede görünen adı verir (distrolarda takma ad olabilir).
func (m Model) displayFor(t tabID, i int) string {
	name := m.targetName(t, i)
	if t == tabDistros {
		return m.displayName(name)
	}
	return name
}

// applyBulk, işaretli satırlar varsa işi toplu hâle getirir.
//
// İşaretli satır yoksa ya da işlem toplu uygulanamıyorsa iş olduğu gibi kalır;
// böylece tek hedefli davranış hiç değişmez.
func (m Model) applyBulk(a action) action {
	if m.markCount(m.active) == 0 || !bulkAllowed[a.kind] {
		return a
	}

	idx := m.markedOrCurrent(m.active)
	if len(idx) < 2 {
		return a
	}

	var targets, labels []string
	for _, i := range idx {
		if name := m.targetName(m.active, i); name != "" {
			targets = append(targets, name)
			labels = append(labels, "  • "+m.displayFor(m.active, i))
		}
	}
	if len(targets) < 2 {
		return a
	}

	a.targets = targets
	a.title = fmt.Sprintf("%s — %d öğe", a.title, len(targets))
	a.body = fmt.Sprintf("Bu işlem %d öğeye sırayla uygulanacak:\n\n%s\n\n"+
		"Bir öğede hata olursa işlem orada durur.",
		len(targets), strings.Join(labels, "\n"))
	a.done = fmt.Sprintf("%d öğe işlendi", len(targets))

	return a
}

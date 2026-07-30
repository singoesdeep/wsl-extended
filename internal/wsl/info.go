package wsl

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// lxssKey, WSL'in distro kayıtlarını tuttuğu registry yolu.
const lxssKey = `Software\Microsoft\Windows\CurrentVersion\Lxss`

// Info, bir distro hakkında toplanabilen tüm bilgidir.
//
// Statik alanlar (registry ve dosya sistemi) distro kapalıyken de okunur.
// Canlı alanlar yalnızca distro çalışırken doldurulur; bilgi almak için
// distroyu başlatmak, kullanıcının istemediği bir yan etki olurdu.
type Info struct {
	Name    string
	GUID    string
	State   State
	Version string
	Default bool

	BasePath   string
	DiskPath   string
	DiskSize   int64
	DefaultUID uint64

	// Canlı alanlar; distro çalışmıyorsa boş kalır.
	Live bool
	// LiveErr, distro çalıştığı hâlde canlı bilgi alınamadığında doludur.
	LiveErr  error
	Kernel   string
	IP       string
	DiskUsed string
	DiskFree string
	DiskUse  string
}

// registryInfo, registry'deki distro kaydını okur.
func registryInfo(name string) (guid, basePath string, uid uint64, err error) {
	root, err := registry.OpenKey(registry.CURRENT_USER, lxssKey, registry.READ)
	if err != nil {
		return "", "", 0, err
	}
	defer root.Close()

	names, err := root.ReadSubKeyNames(-1)
	if err != nil {
		return "", "", 0, err
	}

	for _, sub := range names {
		k, err := registry.OpenKey(root, sub, registry.READ)
		if err != nil {
			continue
		}

		dn, _, err := k.GetStringValue("DistributionName")
		if err != nil || !strings.EqualFold(dn, name) {
			k.Close()
			continue
		}

		bp, _, _ := k.GetStringValue("BasePath")
		u, _, _ := k.GetIntegerValue("DefaultUid")
		k.Close()

		// BasePath "\\?\" ön ekiyle gelebilir; dosya işlemleri için temizlenir.
		return sub, strings.TrimPrefix(bp, `\\?\`), u, nil
	}

	return "", "", 0, nil
}

// Describe, distro hakkındaki bilgileri toplar. Distro çalışmıyorsa yalnızca
// statik alanlar doldurulur ve distro başlatılmaz.
func Describe(ctx context.Context, d Distro) (Info, error) {
	info := Info{
		Name:    d.Name,
		State:   d.State,
		Version: d.Version,
		Default: d.Default,
	}

	guid, basePath, uid, err := registryInfo(d.Name)
	if err == nil {
		info.GUID, info.BasePath, info.DefaultUID = guid, basePath, uid

		if basePath != "" {
			disk := filepath.Join(basePath, "ext4.vhdx")
			if fi, statErr := os.Stat(disk); statErr == nil {
				info.DiskPath, info.DiskSize = disk, fi.Size()
			}
		}
	}

	if !d.IsRunning() {
		return info, nil
	}

	// Her parça `|| true` ile sarılır ve metin işleme distroya bırakılmaz.
	// Fedora'nın WSL imajında awk ve hostname yok; bunlara dayanan bir script
	// 127 ile ölüp bütün canlı bilgiyi düşürüyordu. Ham çıktı alınıp Go
	// tarafında ayrıştırmak, distroda hangi araçların bulunduğundan bağımsız
	// çalışır.
	const script = `{ uname -r || true; }; echo "%%"; ` +
		`{ df -h / || true; }; echo "%%"; ` +
		`{ ip -4 -o addr show scope global 2>/dev/null || true; }`

	out, err := run(ctx, "-d", d.Name, "--exec", "/bin/sh", "-c", script)
	if err != nil {
		// Statik bilgiler geçerli; canlı kısmın neden alınamadığı bildirilir.
		info.LiveErr = err
		return info, nil
	}

	info.Live = true
	parts := strings.Split(out, "%%")
	if len(parts) > 0 {
		info.Kernel = strings.TrimSpace(parts[0])
	}
	if len(parts) > 1 {
		info.DiskUsed, info.DiskFree, info.DiskUse = parseDF(parts[1])
	}
	if len(parts) > 2 {
		info.IP = parseIP(parts[2])
	}

	return info, nil
}

// parseDF, `df -h /` çıktısındaki veri satırından kullanılan/boş/yüzde
// değerlerini çıkarır. Başlık satırı yerelleştirilmiş olabileceği için sütun
// adlarına değil, satırın biçimine bakılır.
func parseDF(out string) (used, free, pct string) {
	for _, line := range splitLines(out) {
		f := strings.Fields(line)
		if len(f) < 5 {
			continue
		}
		// Başlıktaki "Use%" de yüzde işaretiyle biter; ayırt etmek için işaretin
		// önünde sayı olması aranır.
		if !strings.HasSuffix(f[4], "%") {
			continue
		}
		if _, err := strconv.Atoi(strings.TrimSuffix(f[4], "%")); err != nil {
			continue
		}
		return f[2], f[3], f[4]
	}
	return "", "", ""
}

// parseIP, `ip -o addr` çıktısından distronun IPv4 adresini alır.
//
// WSL'de loopback arayüzü de global kapsamda bir adres taşır (10.255.255.254);
// bu ana makineye ait olduğu için atlanır ve gerçek arayüz aranır.
func parseIP(out string) string {
	for _, line := range splitLines(out) {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[1] == "lo" {
			continue
		}

		for i, f := range fields {
			if f != "inet" || i+1 >= len(fields) {
				continue
			}
			addr := fields[i+1]
			if slash := strings.Index(addr, "/"); slash > 0 {
				addr = addr[:slash]
			}
			if addr != "" && !strings.HasPrefix(addr, "127.") {
				return addr
			}
		}
	}
	return ""
}

// OpenExplorer, distronun dosya sistemini Windows Gezgini'nde açar.
//
// Pencere ayrı bir uygulamada açıldığı için komut beklenmez; beklenirse
// arayüz, kullanıcı Gezgin'i kapatana kadar donardı.
func OpenExplorer(distro string) error {
	return exec.Command("explorer.exe", `\\wsl.localhost\`+distro).Start()
}

// OpenVSCode, distroyu VS Code'da uzak oturum olarak açar.
func OpenVSCode(distro string) error {
	// VS Code kurulu değilse komut bulunamaz; hata çağırana bildirilir.
	cmd := exec.Command("code.cmd", "--remote", "wsl+"+distro, "/")
	if _, err := exec.LookPath("code.cmd"); err != nil {
		cmd = exec.Command("code", "--remote", "wsl+"+distro, "/")
	}
	return cmd.Start()
}

// OpenWindow, distroyu ayrı bir konsol penceresinde açar.
func OpenWindow(distro string) error {
	// cmd /c start, wsl.exe'yi yeni bir pencerede başlatır; ilk tırnak
	// pencere başlığı olarak yorumlandığı için boş bir başlık verilir.
	return exec.Command("cmd.exe", "/c", "start", "", "wsl.exe", "-d", distro).Start()
}

// Update, WSL paketini günceller.
func Update(ctx context.Context) error {
	_, err := run(ctx, "--update")
	return err
}

// Status, `wsl --status` çıktısını ham olarak döndürür.
//
// Çıktı yerelleştirilmiş olduğu için ayrıştırılmaz; kullanıcıya olduğu gibi
// gösterilir. Metne göre karar veren bir ayrıştırma dil değişince kırılırdı.
func Status(ctx context.Context) (string, error) {
	return run(ctx, "--status")
}

// Manage sarmalayıcıları: `wsl --manage` altındaki disk işlemleri.

// Resize, distronun sanal diskini verilen boyuta getirir (örn. "60GB").
// Distro kapalı olmalıdır.
func Resize(ctx context.Context, name, size string) error {
	_, err := run(ctx, "--manage", name, "--resize", size)
	return err
}

// SetSparse, seyrek disk kipini açar ya da kapatır. Açıkken silinen dosyaların
// yeri Windows tarafında otomatik geri kazanılır.
func SetSparse(ctx context.Context, name string, on bool) error {
	v := "false"
	if on {
		v = "true"
	}
	_, err := run(ctx, "--manage", name, "--set-sparse", v)
	return err
}

// Move, distronun dosyalarını başka bir konuma taşır. Distro kapalı olmalıdır.
func Move(ctx context.Context, name, location string) error {
	_, err := run(ctx, "--manage", name, "--move", location)
	return err
}

// SetVersion, distronun WSL sürümünü değiştirir (1 ya da 2).
func SetVersion(ctx context.Context, name, version string) error {
	_, err := run(ctx, "--set-version", name, version)
	return err
}

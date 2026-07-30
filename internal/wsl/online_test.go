package wsl

import (
	"strings"
	"testing"
)

// Gerçek `wsl --list --online` çıktısı: başta yerelleştirilmiş açıklama
// satırları, sonra sütun başlığı, sonra katalog.
func TestParseOnlineRealOutput(t *testing.T) {
	out := `Aşağıdakiler, yüklenemedik geçerli dağıtımların listesidir.
'wsl.exe --install <Distro>' kullanarak yükleyin.

NAME                            FRIENDLY NAME
Ubuntu                          Ubuntu
Ubuntu-24.04                    Ubuntu 24.04 LTS
kali-linux                      Kali Linux Rolling
FedoraLinux-44                  Fedora Linux 44
OracleLinux_9_5                 Oracle Linux 9.5
`

	got := parseOnline(out)
	if len(got) != 5 {
		t.Fatalf("5 dağıtım bekleniyordu, %d bulundu: %+v", len(got), got)
	}

	if got[0].Name != "Ubuntu" || got[0].Friendly != "Ubuntu" {
		t.Errorf("ilk kayıt yanlış: %+v", got[0])
	}
	if got[1].Name != "Ubuntu-24.04" || got[1].Friendly != "Ubuntu 24.04 LTS" {
		t.Errorf("boşluklu görünen ad bozuldu: %+v", got[1])
	}
	if got[4].Name != "OracleLinux_9_5" {
		t.Errorf("alt çizgili ad ayrıştırılamadı: %+v", got[4])
	}

	// Açıklama satırları ve başlık katalogda yer almamalı.
	for _, d := range got {
		if strings.EqualFold(d.Name, "NAME") {
			t.Error("sütun başlığı katalog kaydı olarak eklendi")
		}
		if strings.Contains(d.Name, "'") || strings.Contains(d.Name, ",") {
			t.Errorf("açıklama satırı kayıt olarak eklendi: %+v", d)
		}
	}
}

func TestParseOnlineEmpty(t *testing.T) {
	if got := parseOnline(""); len(got) != 0 {
		t.Errorf("boş çıktı için boş liste bekleniyordu: %+v", got)
	}
}

func TestIsDistroID(t *testing.T) {
	valid := []string{"Ubuntu", "Ubuntu-24.04", "OracleLinux_9_5", "kali-linux"}
	for _, s := range valid {
		if !isDistroID(s) {
			t.Errorf("%q geçerli kimlik sayılmalıydı", s)
		}
	}

	invalid := []string{"", "'wsl.exe", "Aşağıdakiler,", "iki kelime", "a:b"}
	for _, s := range invalid {
		if isDistroID(s) {
			t.Errorf("%q geçersiz kimlik sayılmalıydı", s)
		}
	}
}

// Kurulum komutu --no-launch içermeli; olmadan wsl.exe kurulum sonrası
// dağıtımı açıp hesap sorar.
func TestInstallCommandUsesNoLaunch(t *testing.T) {
	cmd := InstallCommand("Ubuntu")

	joined := strings.Join(cmd.Args, " ")
	if !strings.Contains(joined, "--install Ubuntu") {
		t.Errorf("komut yanlış: %q", joined)
	}
	if !strings.Contains(joined, "--no-launch") {
		t.Errorf("--no-launch eksik: %q", joined)
	}
}

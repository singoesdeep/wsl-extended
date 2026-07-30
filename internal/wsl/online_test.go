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

// wsl.exe indirme ilerlemesini taşıma dönüşüyle günceller; yalnızca \n arayan
// bir okuyucu kurulum bitene kadar hiçbir şey göstermez.
func TestScanLinesOrCRSplitsOnCarriageReturn(t *testing.T) {
	data := "İndiriliyor: 10%\rİndiriliyor: 50%\rİndiriliyor: 100%\nKuruldu\n"

	var got []string
	rest := []byte(data)
	for len(rest) > 0 {
		adv, tok, _ := scanLinesOrCR(rest, true)
		if adv == 0 {
			break
		}
		if len(tok) > 0 {
			got = append(got, string(tok))
		}
		rest = rest[adv:]
	}

	if len(got) != 4 {
		t.Fatalf("4 parça bekleniyordu, %d bulundu: %q", len(got), got)
	}
	if got[0] != "İndiriliyor: 10%" {
		t.Errorf("ilk parça = %q", got[0])
	}
	if got[3] != "Kuruldu" {
		t.Errorf("son parça = %q", got[3])
	}
}

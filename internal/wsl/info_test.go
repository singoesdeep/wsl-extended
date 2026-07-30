package wsl

import "testing"

// Fedora'nın gerçek `df -h /` çıktısı. Başlık satırındaki "Use%" da yüzde
// işaretiyle bittiği için biçim kontrolü tek başına yetmez.
func TestParseDFSkipsHeader(t *testing.T) {
	out := "Filesystem      Size  Used Avail Use% Mounted on\n" +
		"/dev/sdd       1007G  725M  955G   1% /\n"

	used, free, pct := parseDF(out)
	if used != "725M" {
		t.Errorf("kullanılan = %q; başlık satırı ayrıştırılmış olabilir", used)
	}
	if free != "955G" {
		t.Errorf("boş = %q", free)
	}
	if pct != "1%" {
		t.Errorf("yüzde = %q", pct)
	}
}

// Başlık yerelleştirilmiş olsa da veri satırı bulunmalı.
func TestParseDFLocalizedHeader(t *testing.T) {
	out := "Dosya sistemi   Boyut Dolu Boş Kull% Bağlanılan yer\n" +
		"/dev/sdd       1007G  725M  955G   1% /\n"

	if used, _, _ := parseDF(out); used != "725M" {
		t.Errorf("kullanılan = %q", used)
	}
}

func TestParseDFEmpty(t *testing.T) {
	if used, free, pct := parseDF(""); used != "" || free != "" || pct != "" {
		t.Errorf("boş çıktı için boş değerler bekleniyordu: %q %q %q", used, free, pct)
	}
}

// WSL'de loopback arayüzü de global kapsamda bir adres taşır; bu adres ana
// makineye aittir ve distronun adresi olarak gösterilmemeli.
func TestParseIPSkipsLoopback(t *testing.T) {
	out := `1: lo    inet 10.255.255.254/32 brd 10.255.255.254 scope global lo\       valid_lft forever preferred_lft forever
2: eth0    inet 172.23.158.119/20 brd 172.23.159.255 scope global eth0\       valid_lft forever preferred_lft forever
`

	if got := parseIP(out); got != "172.23.158.119" {
		t.Errorf("IP = %q; eth0 adresi beklenmişti", got)
	}
}

func TestParseIPEmptyWhenNoInterface(t *testing.T) {
	if got := parseIP(""); got != "" {
		t.Errorf("IP = %q", got)
	}
}

func TestParseIPHandlesMissingIPTool(t *testing.T) {
	// `ip` yoksa çıktı boş ya da hata metni olur; ayrıştırma çökmemeli.
	if got := parseIP("/bin/sh: ip: command not found"); got != "" {
		t.Errorf("IP = %q", got)
	}
}

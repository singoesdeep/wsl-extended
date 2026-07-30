package wsl

import "testing"

func TestParseListRealOutput(t *testing.T) {
	// WSL_UTF8=1 ile alınmış gerçek çıktı.
	out := "  NAME              STATE           VERSION\r\n" +
		"* FedoraLinux-44    Stopped         2\r\n"

	got := parseList(out)
	if len(got) != 1 {
		t.Fatalf("1 distro bekleniyordu, %d bulundu: %+v", len(got), got)
	}

	d := got[0]
	if d.Name != "FedoraLinux-44" {
		t.Errorf("Name = %q", d.Name)
	}
	if d.State != StateStopped {
		t.Errorf("State = %q", d.State)
	}
	if d.Version != "2" {
		t.Errorf("Version = %q", d.Version)
	}
	if !d.Default {
		t.Error("Default true olmalıydı (satır * ile başlıyor)")
	}
}

// Başlık satırı yerelleştirilebildiği için metnine göre değil, son sütunun
// sayı olmamasına göre atlanır.
func TestParseListSkipsLocalizedHeader(t *testing.T) {
	out := "  AD                DURUM           SÜRÜM\n" +
		"* Ubuntu            Running         2\n"

	got := parseList(out)
	if len(got) != 1 {
		t.Fatalf("1 distro bekleniyordu, %d bulundu: %+v", len(got), got)
	}
	if got[0].Name != "Ubuntu" || got[0].State != StateRunning {
		t.Errorf("beklenmeyen kayıt: %+v", got[0])
	}
}

func TestParseListNameWithSpaces(t *testing.T) {
	out := "  NAME    STATE    VERSION\n" +
		"  Ubuntu 22.04 LTS    Stopped    2\n"

	got := parseList(out)
	if len(got) != 1 {
		t.Fatalf("1 distro bekleniyordu, %d bulundu", len(got))
	}
	if got[0].Name != "Ubuntu 22.04 LTS" {
		t.Errorf("Name = %q; boşluklu ad korunmalıydı", got[0].Name)
	}
	if got[0].Default {
		t.Error("Default false olmalıydı")
	}
}

// Tanınmayan durum değeri düşürülmez; ham haliyle saklanır.
func TestParseStateKeepsUnknownValues(t *testing.T) {
	if got := parseState("Converting"); got != State("Converting") {
		t.Errorf("parseState(Converting) = %q", got)
	}
	if got := parseState("running"); got != StateRunning {
		t.Errorf("parseState(running) = %q; normalize edilmeliydi", got)
	}
}

// Arşiv biçimi uzantıdan türetilir; yanlış biçim, içeri aktarılamayan bir
// yedek dosyası üretir.
func TestArchiveFormat(t *testing.T) {
	cases := map[string]string{
		`C:\yedek\a.tar`:     "",
		`C:\yedek\a.tar.gz`:  "tar.gz",
		`C:\yedek\a.TGZ`:     "tar.gz",
		`C:\yedek\a.vhdx`:    "vhd",
		`C:\yedek\a.VHD`:     "vhd",
		`C:\yedek\uzantısız`: "",
	}
	for path, want := range cases {
		if got := archiveFormat(path); got != want {
			t.Errorf("archiveFormat(%q) = %q, %q bekleniyordu", path, got, want)
		}
	}
}

func TestParseListEmpty(t *testing.T) {
	if got := parseList(""); len(got) != 0 {
		t.Errorf("boş çıktı için boş dilim bekleniyordu, %+v bulundu", got)
	}
}

//go:build integration

// Bu testler gerçek wsl.exe'yi çağırır. Varsayılan `go test` koşusuna
// girmezler; `go test -tags integration ./...` ile çalıştırılır.
package wsl

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestListAgainstRealWSL(t *testing.T) {
	if !Available() {
		t.Skip("wsl.exe yok")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	ds, err := List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	for _, d := range ds {
		// UTF-16 çıktısı yanlışlıkla geri gelirse adlar NUL byte içerir;
		// bu kontrol WSL_UTF8 düzeltmesinin çalıştığını doğrular.
		for _, r := range d.Name {
			if r == 0 {
				t.Fatalf("distro adında NUL byte var (%q): WSL_UTF8 uygulanmamış", d.Name)
			}
		}
		if d.Name == "" {
			t.Error("boş distro adı ayrıştırıldı")
		}
		if d.Version == "" {
			t.Errorf("%s: sürüm ayrıştırılamadı", d.Name)
		}
		t.Logf("distro: %+v", d)
	}
}

// Describe, registry ve dosya sisteminden statik bilgileri okur; distroyu
// başlatmaz.
func TestDescribeAgainstRealWSL(t *testing.T) {
	if !Available() {
		t.Skip("wsl.exe yok")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ds, err := List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(ds) == 0 {
		t.Skip("kurulu distro yok")
	}

	info, err := Describe(ctx, ds[0])
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}

	if info.GUID == "" {
		t.Error("registry'den GUID okunamadı")
	}
	if info.BasePath == "" {
		t.Error("registry'den BasePath okunamadı")
	}
	if info.DiskPath != "" && info.DiskSize == 0 {
		t.Error("disk dosyası bulundu ama boyutu okunamadı")
	}
	t.Logf("info: %+v", info)
}

func TestListOnlineAgainstRealWSL(t *testing.T) {
	if !Available() {
		t.Skip("wsl.exe yok")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	list, err := ListOnline(ctx)
	if err != nil {
		t.Fatalf("ListOnline: %v", err)
	}
	if len(list) == 0 {
		t.Fatal("katalog boş döndü")
	}

	for _, d := range list {
		if d.Name == "" || d.Friendly == "" {
			t.Errorf("eksik kayıt: %+v", d)
		}
		if strings.ContainsAny(d.Name, " ',") {
			t.Errorf("açıklama satırı kayıt olarak ayrıştırıldı: %+v", d)
		}
	}
	t.Logf("%d kurulabilir dağıtım, ilki: %+v", len(list), list[0])
}

func TestVersionAgainstRealWSL(t *testing.T) {
	if !Available() {
		t.Skip("wsl.exe yok")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	v, err := Version(ctx)
	if err != nil {
		t.Fatalf("Version: %v", err)
	}
	if v == "" {
		t.Error("sürüm boş döndü")
	}
	t.Logf("WSL sürümü: %s", v)
}

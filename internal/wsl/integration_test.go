//go:build integration

// Bu testler gerçek wsl.exe'yi çağırır. Varsayılan `go test` koşusuna
// girmezler; `go test -tags integration ./...` ile çalıştırılır.
package wsl

import (
	"context"
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

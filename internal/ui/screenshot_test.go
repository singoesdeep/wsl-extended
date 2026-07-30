package ui

import (
	"os"
	"testing"

	"github.com/singoesdeep/wsl-extended/internal/wsl"
)

// TestWriteScreenshot, README'de kullanılan örnek ekranı üretir.
//
// Elle yazılmış bir örnek zamanla gerçeğinden ayrışır; bu çıktı doğrudan
// View'dan geldiği için arayüz değişince yeniden üretilebilir.
// Yalnızca WSLX_SCREENSHOT ayarlıyken dosyaya yazar.
func TestWriteScreenshot(t *testing.T) {
	path := os.Getenv("WSLX_SCREENSHOT")
	if path == "" {
		t.Skip("WSLX_SCREENSHOT ayarlı değil")
	}

	m := New()
	m.wslOK, m.wslcOK = true, true
	m.width, m.height = 92, 20
	m.wslVersion = "2.9.4.0"
	m.distros = []wsl.Distro{
		{Name: "FedoraLinux-44", State: wsl.StateRunning, Version: "2", Default: true},
		{Name: "openSUSE-Tumbleweed", State: wsl.StateStopped, Version: "2"},
		{Name: "kali-linux", State: wsl.StateStopped, Version: "2"},
	}
	m.cursors[tabDistros] = 0

	if err := os.WriteFile(path, []byte(m.View()), 0o600); err != nil {
		t.Fatalf("yazılamadı: %v", err)
	}
	t.Logf("ekran çıktısı yazıldı: %s", path)
}

// Package store, uygulamanın kendi kalıcı verisini tutar.
//
// Şu an yalnızca distro takma adlarını saklar. Takma ad, WSL'in gerçek distro
// adını değiştirmez: WSL'de yeniden adlandırma komutu yoktur ve registry'ye
// dokunmak distroyu görünmez kılma riski taşır. Bunun yerine görünen ad bu
// dosyada tutulur, komutlar her zaman gerçek adla çalıştırılır.
package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Data, diske yazılan yapıdır.
type Data struct {
	// Aliases: gerçek distro adı → kullanıcının verdiği görünen ad.
	Aliases map[string]string `json:"aliases"`
}

// Path, veri dosyasının konumu.
func Path() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		base = home
	}
	return filepath.Join(base, "wsl-extended", "data.json")
}

// Load, veriyi okur. Dosya yoksa boş veri döner; bu bir hata değildir.
func Load(path string) (*Data, error) {
	d := &Data{Aliases: map[string]string{}}

	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return d, nil
		}
		return d, err
	}

	if err := json.Unmarshal(raw, d); err != nil {
		// Bozuk veri yüzünden uygulama açılmamazlık etmemeli; boş veriyle
		// devam edilir ve dosya ilk kayıtta yenilenir.
		return &Data{Aliases: map[string]string{}}, err
	}
	if d.Aliases == nil {
		d.Aliases = map[string]string{}
	}
	return d, nil
}

// Save, veriyi diske yazar.
func Save(path string, d *Data) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	raw, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0o600)
}

// Alias, gerçek ad için görünen adı döndürür. Takma ad yoksa gerçek ad döner.
func (d *Data) Alias(real string) string {
	if d == nil {
		return real
	}
	if a, ok := d.Aliases[real]; ok && strings.TrimSpace(a) != "" {
		return a
	}
	return real
}

// HasAlias, distronun takma adı olup olmadığını söyler.
func (d *Data) HasAlias(real string) bool {
	if d == nil {
		return false
	}
	a, ok := d.Aliases[real]
	return ok && strings.TrimSpace(a) != ""
}

// SetAlias, takma adı belirler. Boş ad takma adı kaldırır, yani liste gerçek
// ada geri döner.
func (d *Data) SetAlias(real, alias string) {
	if d.Aliases == nil {
		d.Aliases = map[string]string{}
	}

	alias = strings.TrimSpace(alias)
	if alias == "" || alias == real {
		delete(d.Aliases, real)
		return
	}
	d.Aliases[real] = alias
}

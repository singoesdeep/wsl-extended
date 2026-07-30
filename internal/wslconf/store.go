package wslconf

import (
	"os"
	"path/filepath"
)

// WSLConfigPath, Windows tarafındaki genel yapılandırma dosyasının yolu.
func WSLConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".wslconfig"
	}
	return filepath.Join(home, ".wslconfig")
}

// LoadFile, dosyayı okur. Dosya yoksa boş bir yapılandırma döner: henüz
// oluşturulmamış bir `.wslconfig` düzenlenebilir olmalıdır.
func LoadFile(path string) (*File, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Parse(""), nil
		}
		return nil, err
	}
	return Parse(string(data)), nil
}

// SaveFile, içeriği diske yazar ve varsa önceki hâlini `.bak` uzantısıyla
// saklar. Yedek, yanlış bir düzenlemeden dönebilmek için alınır.
func SaveFile(path, content string) error {
	if old, err := os.ReadFile(path); err == nil {
		if err := os.WriteFile(path+".bak", old, 0o600); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o600)
}

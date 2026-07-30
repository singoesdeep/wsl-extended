package wslconf

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Field, düzenlenebilir tek bir yapılandırma anahtarını tanımlar.
type Field struct {
	Section string
	Key     string
	Label   string
	Hint    string

	// Validate, boş olmayan değerleri denetler. Boş değer her zaman geçerlidir
	// ve anahtarın dosyadan silinmesi anlamına gelir.
	Validate func(string) error
}

// Check, alanın değerini doğrular. Boş değer "ayarlama" demektir.
func (f Field) Check(value string) error {
	if strings.TrimSpace(value) == "" || f.Validate == nil {
		return nil
	}
	return f.Validate(value)
}

// sizePattern, WSL'in kabul ettiği bellek boyutu yazımı: 8GB, 512MB, 2048...
var sizePattern = regexp.MustCompile(`(?i)^\d+(\.\d+)?\s*(b|kb|mb|gb|tb)?$`)

func validateSize(v string) error {
	if !sizePattern.MatchString(strings.TrimSpace(v)) {
		return fmt.Errorf("boyut biçimi geçersiz (örnek: 8GB, 512MB)")
	}
	return nil
}

// CheckSize, bellek ya da disk boyutu yazımını doğrular (8GB, 512MB gibi).
func CheckSize(v string) error { return validateSize(v) }

func validateInt(v string) error {
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return fmt.Errorf("tam sayı olmalı")
	}
	if n <= 0 {
		return fmt.Errorf("sıfırdan büyük olmalı")
	}
	return nil
}

func validateBool(v string) error {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "true", "false":
		return nil
	}
	return fmt.Errorf("true ya da false olmalı")
}

func oneOf(options ...string) func(string) error {
	return func(v string) error {
		for _, o := range options {
			if strings.EqualFold(strings.TrimSpace(v), o) {
				return nil
			}
		}
		return fmt.Errorf("şunlardan biri olmalı: %s", strings.Join(options, ", "))
	}
}

// WSLConfigFields, `%UserProfile%\.wslconfig` içindeki en çok kullanılan
// ayarlardır. Bu dosya tüm WSL2 dağıtımları için geçerlidir.
func WSLConfigFields() []Field {
	return []Field{
		{Section: "wsl2", Key: "memory", Label: "Bellek üst sınırı",
			Hint: "8GB — boş bırakılırsa WSL kendi varsayılanını kullanır",
			Validate: validateSize},
		{Section: "wsl2", Key: "processors", Label: "İşlemci sayısı",
			Hint: "4", Validate: validateInt},
		{Section: "wsl2", Key: "swap", Label: "Takas alanı",
			Hint: "2GB — 0 yazmak takası kapatır", Validate: validateSize},
		{Section: "wsl2", Key: "networkingMode", Label: "Ağ kipi",
			Hint: "nat ya da mirrored", Validate: oneOf("nat", "mirrored")},
		{Section: "wsl2", Key: "localhostForwarding", Label: "localhost yönlendirme",
			Hint: "true / false", Validate: validateBool},
		{Section: "wsl2", Key: "autoMemoryReclaim", Label: "Belleği otomatik geri al",
			Hint: "disabled, gradual ya da dropcache",
			Validate: oneOf("disabled", "gradual", "dropcache")},
		{Section: "wsl2", Key: "nestedVirtualization", Label: "İç içe sanallaştırma",
			Hint: "true / false", Validate: validateBool},
	}
}

// DistroConfFields, distro içindeki `/etc/wsl.conf` ayarlarıdır. Bu dosya
// yalnızca ilgili distroyu etkiler.
func DistroConfFields() []Field {
	return []Field{
		{Section: "boot", Key: "systemd", Label: "systemd",
			Hint: "true / false", Validate: validateBool},
		{Section: "user", Key: "default", Label: "Varsayılan kullanıcı",
			Hint: "kullanıcı adı"},
		{Section: "automount", Key: "enabled", Label: "Windows sürücülerini bağla",
			Hint: "true / false", Validate: validateBool},
		{Section: "automount", Key: "options", Label: "Bağlama seçenekleri",
			Hint: "metadata,umask=22,fmask=11"},
		{Section: "interop", Key: "enabled", Label: "Windows uygulamalarını çalıştır",
			Hint: "true / false", Validate: validateBool},
		{Section: "interop", Key: "appendWindowsPath", Label: "Windows PATH'ini ekle",
			Hint: "true / false", Validate: validateBool},
		{Section: "network", Key: "generateResolvConf", Label: "resolv.conf üret",
			Hint: "true / false", Validate: validateBool},
		{Section: "network", Key: "hostname", Label: "Makine adı", Hint: "fedora"},
	}
}

// Apply, form değerlerini dosyaya uygular. Boş bırakılan alanlar dosyadan
// silinir, böylece WSL kendi varsayılanına döner.
func Apply(f *File, fields []Field, values []string) {
	for i, field := range fields {
		if i >= len(values) {
			break
		}
		v := strings.TrimSpace(values[i])
		if v == "" {
			f.Unset(field.Section, field.Key)
			continue
		}
		f.Set(field.Section, field.Key, v)
	}
}

// Values, dosyadaki mevcut değerleri form sırasına göre okur.
func Values(f *File, fields []Field) []string {
	out := make([]string, len(fields))
	for i, field := range fields {
		if v, ok := f.Get(field.Section, field.Key); ok {
			out[i] = v
		}
	}
	return out
}

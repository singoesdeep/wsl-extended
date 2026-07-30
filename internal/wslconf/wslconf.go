// Package wslconf, WSL'in INI biçimli yapılandırma dosyalarını
// (`.wslconfig` ve `wsl.conf`) düzenler.
//
// Dosya bir satır dizisi olarak tutulur ve düzenleme yerinde yapılır: yorumlar,
// boş satırlar, satır sırası ve tanımadığımız anahtarlar olduğu gibi korunur.
// Ayrıştırıp yeniden üreten bir yaklaşım, kullanıcının dosyasındaki açıklamaları
// ve elle eklediği ayarları sessizce silerdi.
package wslconf

import (
	"strings"
)

// File, düzenlenebilir bir INI dosyasıdır.
type File struct {
	lines []string
	// crlf, dosyanın Windows satır sonu kullanıp kullanmadığını hatırlar;
	// kaydederken aynı biçim korunur.
	crlf bool
}

// Parse, dosya içeriğini düzenlenebilir hâle getirir. Boş içerik geçerlidir:
// henüz var olmayan bir yapılandırma dosyası bu şekilde oluşturulur.
func Parse(data string) *File {
	// Windows editörleri dosyayı UTF-8 BOM ile kaydedebilir. BOM temizlenmezse
	// ilk satır görünmez bir karakterle başlar, bölüm başlığı tanınmaz ve
	// dosyadaki tüm ayarlar okunamaz hâle gelir.
	data = strings.TrimPrefix(data, "\ufeff")

	f := &File{crlf: strings.Contains(data, "\r\n")}

	normalized := strings.ReplaceAll(data, "\r\n", "\n")
	if normalized == "" {
		return f
	}
	f.lines = strings.Split(strings.TrimSuffix(normalized, "\n"), "\n")
	return f
}

// sectionName, satır bir bölüm başlığıysa adını döndürür.
func sectionName(line string) (string, bool) {
	t := strings.TrimSpace(line)
	if len(t) >= 2 && strings.HasPrefix(t, "[") && strings.HasSuffix(t, "]") {
		return strings.TrimSpace(t[1 : len(t)-1]), true
	}
	return "", false
}

// keyOf, satır bir anahtar ataması ise anahtarı döndürür. Yorum satırları
// (# ya da ;) atama sayılmaz.
func keyOf(line string) (string, bool) {
	t := strings.TrimSpace(line)
	if t == "" || strings.HasPrefix(t, "#") || strings.HasPrefix(t, ";") {
		return "", false
	}
	i := strings.Index(t, "=")
	if i <= 0 {
		return "", false
	}
	return strings.TrimSpace(t[:i]), true
}

// find, verilen bölüm ve anahtara ait satırın indeksini döndürür.
func (f *File) find(section, key string) int {
	current := ""
	for i, line := range f.lines {
		if s, ok := sectionName(line); ok {
			current = s
			continue
		}
		if !strings.EqualFold(current, section) {
			continue
		}
		if k, ok := keyOf(line); ok && strings.EqualFold(k, key) {
			return i
		}
	}
	return -1
}

// Get, anahtarın değerini döndürür.
func (f *File) Get(section, key string) (string, bool) {
	i := f.find(section, key)
	if i < 0 {
		return "", false
	}
	_, v, _ := strings.Cut(f.lines[i], "=")
	return strings.TrimSpace(v), true
}

// Set, anahtarı yerinde günceller; yoksa ilgili bölümün sonuna ekler. Bölüm de
// yoksa dosyanın sonunda oluşturulur.
func (f *File) Set(section, key, value string) {
	if i := f.find(section, key); i >= 0 {
		f.lines[i] = key + "=" + value
		return
	}

	// Bölümün son satırını bul: yeni anahtar oraya eklenir.
	sectionStart, insertAt := -1, -1
	current := ""
	for i, line := range f.lines {
		if s, ok := sectionName(line); ok {
			if strings.EqualFold(current, section) && sectionStart >= 0 {
				break // bölüm bitti
			}
			current = s
			if strings.EqualFold(s, section) {
				sectionStart, insertAt = i, i
			}
			continue
		}
		if strings.EqualFold(current, section) && sectionStart >= 0 {
			// Sondaki boş satırların üstüne eklemek için yalnızca dolu
			// satırlarda ilerlenir.
			if strings.TrimSpace(line) != "" {
				insertAt = i
			}
		}
	}

	if sectionStart < 0 {
		if len(f.lines) > 0 && strings.TrimSpace(f.lines[len(f.lines)-1]) != "" {
			f.lines = append(f.lines, "")
		}
		f.lines = append(f.lines, "["+section+"]", key+"="+value)
		return
	}

	line := key + "=" + value
	f.lines = append(f.lines[:insertAt+1], append([]string{line}, f.lines[insertAt+1:]...)...)
}

// Unset, anahtar satırını siler. Boş bırakılan bir alan, dosyada değeri
// olmayan bir anahtar yerine hiç anahtar bulunmaması anlamına gelir; böylece
// WSL kendi varsayılanını uygular.
func (f *File) Unset(section, key string) {
	if i := f.find(section, key); i >= 0 {
		f.lines = append(f.lines[:i], f.lines[i+1:]...)
	}
}

// String, dosyanın kaydedilecek hâlini üretir.
func (f *File) String() string {
	if len(f.lines) == 0 {
		return ""
	}
	sep := "\n"
	if f.crlf {
		sep = "\r\n"
	}
	return strings.Join(f.lines, sep) + sep
}

// Lines, karşılaştırma için satırların kopyasını döndürür.
func (f *File) Lines() []string {
	out := make([]string, len(f.lines))
	copy(out, f.lines)
	return out
}

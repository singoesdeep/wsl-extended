package wslconf

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Düzenleme, kullanıcının yorumlarını ve tanımadığımız anahtarlarını
// silmemeli; dosya kullanıcının, biz yalnızca ziyaretçiyiz.
func TestSetPreservesCommentsAndUnknownKeys(t *testing.T) {
	src := `# Bellek ayarı - dokunma
[wsl2]
memory=4GB
; noktalı virgüllü yorum
kernelCommandLine=quiet
processors=2
`

	f := Parse(src)
	f.Set("wsl2", "memory", "8GB")
	got := f.String()

	for _, want := range []string{
		"# Bellek ayarı - dokunma",
		"; noktalı virgüllü yorum",
		"kernelCommandLine=quiet",
		"processors=2",
		"memory=8GB",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("çıktıda %q yok:\n%s", want, got)
		}
	}
	if strings.Contains(got, "memory=4GB") {
		t.Errorf("eski değer kaldı:\n%s", got)
	}
}

// Windows editörleriyle kaydedilmiş dosyalar UTF-8 BOM taşıyabilir; BOM
// temizlenmezse ilk bölüm başlığı tanınmaz ve tüm ayarlar görünmez olur.
func TestParseStripsUTF8BOM(t *testing.T) {
	f := Parse("\ufeff[wsl2]\nmemory=8GB\n")

	v, ok := f.Get("wsl2", "memory")
	if !ok {
		t.Fatal("BOM'lu dosyada bölüm tanınmadı")
	}
	if v != "8GB" {
		t.Errorf("değer = %q", v)
	}
}

func TestSetAddsKeyToExistingSection(t *testing.T) {
	f := Parse("[wsl2]\nmemory=4GB\n")
	f.Set("wsl2", "swap", "2GB")

	got := f.String()
	if !strings.Contains(got, "swap=2GB") {
		t.Errorf("anahtar eklenmedi:\n%s", got)
	}
	// Yeni anahtar kendi bölümünde kalmalı.
	if strings.Index(got, "swap=2GB") < strings.Index(got, "[wsl2]") {
		t.Errorf("anahtar bölümün dışına yazıldı:\n%s", got)
	}
}

func TestSetCreatesMissingSection(t *testing.T) {
	f := Parse("")
	f.Set("wsl2", "memory", "8GB")

	got := f.String()
	if !strings.Contains(got, "[wsl2]") || !strings.Contains(got, "memory=8GB") {
		t.Errorf("bölüm oluşturulmadı:\n%s", got)
	}
}

// Aynı anahtar adı başka bölümde varsa yanlış olan güncellenmemeli.
func TestSetTargetsCorrectSection(t *testing.T) {
	f := Parse("[boot]\nenabled=true\n\n[interop]\nenabled=false\n")
	f.Set("interop", "enabled", "true")

	got := f.String()
	boot := got[strings.Index(got, "[boot]"):strings.Index(got, "[interop]")]
	if !strings.Contains(boot, "enabled=true") {
		t.Errorf("[boot] bölümü bozuldu:\n%s", got)
	}
	interop := got[strings.Index(got, "[interop]"):]
	if !strings.Contains(interop, "enabled=true") {
		t.Errorf("[interop] güncellenmedi:\n%s", got)
	}
}

func TestGetIsCaseInsensitive(t *testing.T) {
	f := Parse("[WSL2]\nMemory = 8GB\n")

	v, ok := f.Get("wsl2", "memory")
	if !ok {
		t.Fatal("anahtar bulunamadı")
	}
	if v != "8GB" {
		t.Errorf("değer = %q", v)
	}
}

// Yorum satırındaki "anahtar=değer" gerçek atama sayılmamalı.
func TestCommentedKeyIsNotFound(t *testing.T) {
	f := Parse("[wsl2]\n#memory=4GB\n")
	if _, ok := f.Get("wsl2", "memory"); ok {
		t.Error("yorum satırı atama olarak okundu")
	}
}

func TestUnsetRemovesLine(t *testing.T) {
	f := Parse("[wsl2]\nmemory=4GB\nswap=2GB\n")
	f.Unset("wsl2", "memory")

	got := f.String()
	if strings.Contains(got, "memory") {
		t.Errorf("satır silinmedi:\n%s", got)
	}
	if !strings.Contains(got, "swap=2GB") {
		t.Errorf("yanlış satır silindi:\n%s", got)
	}
}

// Windows satır sonu kullanan bir dosya, kaydedildiğinde biçimini korumalı.
func TestPreservesCRLF(t *testing.T) {
	f := Parse("[wsl2]\r\nmemory=4GB\r\n")
	f.Set("wsl2", "memory", "8GB")

	if !strings.Contains(f.String(), "\r\n") {
		t.Error("CRLF satır sonu kaybedildi")
	}
}

func TestApplyEmptyValueRemovesKey(t *testing.T) {
	f := Parse("[wsl2]\nmemory=4GB\nprocessors=2\n")
	fields := []Field{
		{Section: "wsl2", Key: "memory"},
		{Section: "wsl2", Key: "processors"},
	}

	Apply(f, fields, []string{"", "4"})

	got := f.String()
	if strings.Contains(got, "memory") {
		t.Errorf("boş değer anahtarı silmedi:\n%s", got)
	}
	if !strings.Contains(got, "processors=4") {
		t.Errorf("değer güncellenmedi:\n%s", got)
	}
}

func TestValuesReadsExisting(t *testing.T) {
	f := Parse("[wsl2]\nmemory=8GB\n")
	fields := []Field{
		{Section: "wsl2", Key: "memory"},
		{Section: "wsl2", Key: "swap"},
	}

	got := Values(f, fields)
	if got[0] != "8GB" {
		t.Errorf("values[0] = %q", got[0])
	}
	if got[1] != "" {
		t.Errorf("values[1] = %q; ayarlanmamış alan boş olmalı", got[1])
	}
}

func TestFieldValidation(t *testing.T) {
	fields := WSLConfigFields()
	byKey := map[string]Field{}
	for _, f := range fields {
		byKey[f.Key] = f
	}

	cases := []struct {
		key     string
		value   string
		wantErr bool
	}{
		{"memory", "8GB", false},
		{"memory", "512MB", false},
		{"memory", "çok", true},
		{"processors", "4", false},
		{"processors", "0", true},
		{"processors", "iki", true},
		{"localhostForwarding", "true", false},
		{"localhostForwarding", "evet", true},
		{"networkingMode", "mirrored", false},
		{"networkingMode", "bridge", true},
		{"autoMemoryReclaim", "gradual", false},
		{"autoMemoryReclaim", "kapalı", true},
		// Boş değer her zaman geçerli: anahtar silinir.
		{"memory", "", false},
		{"processors", "", false},
	}

	for _, c := range cases {
		f, ok := byKey[c.key]
		if !ok {
			t.Fatalf("%s alanı tanımlı değil", c.key)
		}
		err := f.Check(c.value)
		if c.wantErr && err == nil {
			t.Errorf("%s=%q kabul edildi, reddedilmeliydi", c.key, c.value)
		}
		if !c.wantErr && err != nil {
			t.Errorf("%s=%q reddedildi: %v", c.key, c.value, err)
		}
	}
}

func TestDiffDetectsChange(t *testing.T) {
	old := []string{"[wsl2]", "memory=4GB", "processors=2"}
	new := []string{"[wsl2]", "memory=8GB", "processors=2"}

	d := Diff(old, new)
	if !HasChanges(d) {
		t.Fatal("fark bulunamadı")
	}

	out := FormatDiff(d, 1)
	if !strings.Contains(out, "- memory=4GB") {
		t.Errorf("silinen satır yok:\n%s", out)
	}
	if !strings.Contains(out, "+ memory=8GB") {
		t.Errorf("eklenen satır yok:\n%s", out)
	}
	if strings.Contains(out, "+ processors") || strings.Contains(out, "- processors") {
		t.Errorf("değişmeyen satır değişmiş gösterildi:\n%s", out)
	}
}

func TestDiffNoChanges(t *testing.T) {
	lines := []string{"[wsl2]", "memory=4GB"}
	if HasChanges(Diff(lines, lines)) {
		t.Error("aynı içerik için değişiklik bildirildi")
	}
}

// Kaydetme, önceki hâli .bak olarak saklamalı; yanlış düzenlemeden dönüş yolu.
func TestSaveFileWritesBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".wslconfig")

	if err := os.WriteFile(path, []byte("[wsl2]\nmemory=4GB\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := SaveFile(path, "[wsl2]\nmemory=8GB\n"); err != nil {
		t.Fatalf("SaveFile: %v", err)
	}

	bak, err := os.ReadFile(path + ".bak")
	if err != nil {
		t.Fatalf("yedek oluşturulmadı: %v", err)
	}
	if !strings.Contains(string(bak), "memory=4GB") {
		t.Errorf("yedek eski içeriği tutmuyor: %q", bak)
	}

	cur, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cur), "memory=8GB") {
		t.Errorf("yeni içerik yazılmadı: %q", cur)
	}
}

// Dosya henüz yoksa kaydetme çalışmalı ve yedek aramamalı.
func TestSaveFileCreatesNew(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".wslconfig")

	if err := SaveFile(path, "[wsl2]\nmemory=8GB\n"); err != nil {
		t.Fatalf("SaveFile: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("dosya oluşturulmadı: %v", err)
	}
	if _, err := os.Stat(path + ".bak"); !os.IsNotExist(err) {
		t.Error("olmayan dosya için yedek oluşturuldu")
	}
}

func TestLoadFileMissingIsEmpty(t *testing.T) {
	f, err := LoadFile(filepath.Join(t.TempDir(), "yok.wslconfig"))
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if f.String() != "" {
		t.Errorf("boş içerik bekleniyordu: %q", f.String())
	}
}

// Düzenleme döngüsü kayıpsız olmalı: oku, değiştir, yaz, tekrar oku.
func TestRoundTrip(t *testing.T) {
	src := "# başlık\n[wsl2]\nmemory=4GB\n\n[experimental]\nsparseVhd=true\n"

	f := Parse(src)
	f.Set("wsl2", "processors", "4")
	out := f.String()

	again := Parse(out)
	if v, _ := again.Get("wsl2", "processors"); v != "4" {
		t.Errorf("processors = %q", v)
	}
	if v, _ := again.Get("experimental", "sparseVhd"); v != "true" {
		t.Errorf("diğer bölüm bozuldu: %q", v)
	}
	if !strings.Contains(out, "# başlık") {
		t.Error("yorum kayboldu")
	}
}

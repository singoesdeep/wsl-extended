package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAliasFallsBackToRealName(t *testing.T) {
	d := &Data{Aliases: map[string]string{}}

	if got := d.Alias("FedoraLinux-44"); got != "FedoraLinux-44" {
		t.Errorf("Alias = %q; takma ad yokken gerçek ad dönmeliydi", got)
	}
	if d.HasAlias("FedoraLinux-44") {
		t.Error("HasAlias true döndü")
	}
}

func TestSetAndGetAlias(t *testing.T) {
	d := &Data{}
	d.SetAlias("FedoraLinux-44", "iş makinesi")

	if got := d.Alias("FedoraLinux-44"); got != "iş makinesi" {
		t.Errorf("Alias = %q", got)
	}
	if !d.HasAlias("FedoraLinux-44") {
		t.Error("HasAlias false döndü")
	}
}

// Boş ad takma adı kaldırmalı: liste gerçek ada döner.
func TestEmptyAliasRemovesEntry(t *testing.T) {
	d := &Data{}
	d.SetAlias("a", "takma")
	d.SetAlias("a", "   ")

	if d.HasAlias("a") {
		t.Error("boşluk takma ad olarak saklandı")
	}
	if got := d.Alias("a"); got != "a" {
		t.Errorf("Alias = %q", got)
	}
}

// Gerçek adın kendisi takma ad olarak verilirse kayıt tutulmamalı.
func TestAliasEqualToRealIsNotStored(t *testing.T) {
	d := &Data{}
	d.SetAlias("a", "a")

	if len(d.Aliases) != 0 {
		t.Errorf("gereksiz kayıt tutuldu: %v", d.Aliases)
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "alt", "data.json")

	d := &Data{}
	d.SetAlias("FedoraLinux-44", "iş makinesi")
	if err := Save(path, d); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Alias("FedoraLinux-44") != "iş makinesi" {
		t.Errorf("takma ad kaybedildi: %v", got.Aliases)
	}
}

func TestLoadMissingFileIsEmpty(t *testing.T) {
	d, err := Load(filepath.Join(t.TempDir(), "yok.json"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(d.Aliases) != 0 {
		t.Errorf("boş veri bekleniyordu: %v", d.Aliases)
	}
}

// Bozuk veri dosyası uygulamayı açılamaz hâle getirmemeli.
func TestLoadCorruptFileReturnsUsableData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	if err := os.WriteFile(path, []byte("{bozuk"), 0o600); err != nil {
		t.Fatal(err)
	}

	d, err := Load(path)
	if err == nil {
		t.Error("bozuk dosya için hata bildirilmedi")
	}
	if d == nil || d.Aliases == nil {
		t.Fatal("kullanılabilir veri dönmedi")
	}
	// Boş veriyle çalışmaya devam edebilmeli.
	d.SetAlias("a", "b")
	if d.Alias("a") != "b" {
		t.Error("bozuk dosyadan sonra veri kullanılamıyor")
	}
}

// nil Data ile çağrılar panik atmamalı.
func TestNilDataIsSafe(t *testing.T) {
	var d *Data
	if got := d.Alias("a"); got != "a" {
		t.Errorf("Alias = %q", got)
	}
	if d.HasAlias("a") {
		t.Error("HasAlias true döndü")
	}
}

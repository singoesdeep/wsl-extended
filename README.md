# wsl-extended

WSL distrolarını ve `wslc` kapsayıcılarını tek bir terminal arayüzünden yönetmek için Go ile yazılmış TUI.

Sekmeli yapı: **Distros · Containers · Images · Volumes · Networks**

## Durum

**Faz 3 tamam.** Listeler, aksiyonlar (onay kapısıyla), canlı günlük akışı, kaynak
kullanımı paneli ve tek tuşla kabuk. Sırada Faz 4 var: distro yedekleme (export/import).

Günlük ve kabuk özellikleri gerçek bir kapsayıcıyla **henüz denenmedi** — bu makinede
hiç kapsayıcı yok. Komut sözdizimleri `wslc --help` çıktısına göre yazıldı ve akış
mantığı testlerle doğrulandı, ama ilk kapsayıcını oluşturduğunda bu iki özelliği
gözden geçirmek gerekir.

Yol haritası ve mimari kararlar: [docs/PLAN.md](docs/PLAN.md)

## Gereksinimler

- Go 1.26+
- WSL 2.9+ (`wsl.exe`)
- `wslc.exe` — kapsayıcı sekmeleri için. Yoksa uygulama çalışır, o sekmeler bilgi ekranı gösterir.

## Çalıştırma

```bash
go run ./cmd/wsl-extended
```

Derlemek için:

```bash
go build -o bin/wsl-extended.exe ./cmd/wsl-extended
```

## Tuşlar

| Tuş | İşlev |
|---|---|
| `Tab` / `Shift+Tab`, `h` / `l` | Sekme değiştir |
| `1`–`5` | Doğrudan sekmeye git |
| `j` / `k`, ok tuşları | Satır gezinme |
| `g` / `G` | Listenin başı / sonu |
| `Enter` | Kabuğa gir (distro veya çalışan kapsayıcı) |
| `L` | Canlı günlük paneli (kapsayıcı) |
| `t` | Kaynak kullanımı paneli |
| `s` / `S` | Başlat / durdur (distro veya kapsayıcı) |
| `K` | Kapsayıcıyı sonlandır (kill) |
| `u` | Distroyu varsayılan yap |
| `d` | Sil |
| `X` | Tüm WSL'i kapat |
| `r` | Yenile |
| `?` | Yardımı genişlet |
| `q` | Çık |

## Onay davranışı

Durum değiştiren her işlem onay ister. İki kip vardır:

- **y/n** — geri alınabilir işlemler: başlat, durdur, kill, kapsayıcı/imaj/ağ silme.
- **Adı yazma** — geri dönüşü olmayan işlemler: `wsl --unregister` ve birim silme.
  Hedefin adı harfi harfine yazılmadan `enter` çalışmaz, ve bu kipte `y` bir onay
  tuşu değil yazılan metnin parçasıdır.

Onay açıkken tuşlar arkadaki listeye ulaşmaz ve otomatik yenileme durur; böylece
onay beklerken imleç kayıp işlem yanlış hedefe uygulanamaz.

## Testler

```bash
go test ./...
```

Gerçek `wsl.exe` ve `wslc.exe` çağıran entegrasyon testleri ayrı bir etikettedir.
Yalnızca okuma yaparlar; hiçbir distro, kapsayıcı ya da birim oluşturmaz veya silmezler:

```bash
go test -tags integration -v ./...
```

## Mimari notu

`internal/wsl` ve `internal/wslc` paketleri Bubble Tea'yi bilmez; saf Go tipleri ve
`error` döndürür. Arayüz katmanı bunları `tea.Cmd` içine sarar. Böylece CLI
sarmalayıcıları TUI olmadan test edilebilir.

İki tuzak koda gömülü olarak çözülmüştür: `wsl.exe` çıktısını UTF-16LE yazdığı için
her çağrıya `WSL_UTF8=1` geçilir, ve CLI çıktıları yerelleştirilmiş olabildiğinden
hiçbir karar hata metnine bakarak verilmez.

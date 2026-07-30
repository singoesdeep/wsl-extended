# wsl-extended

WSL distrolarını ve `wslc` kapsayıcılarını tek bir terminal arayüzünden yönetmek için Go ile yazılmış TUI.

Sekmeli yapı: **Distros · Containers · Images · Volumes · Networks**

## Durum

**Faz 2 tamam — aksiyonlar onay kapısıyla birlikte çalışıyor.** Distro başlat/durdur/
varsayılan yap/sil, kapsayıcı başlat/durdur/sonlandır/sil, imaj-birim-ağ silme.
Sırada Faz 3 var: canlı log akışı, `stats` ve tek tuşla shell.

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

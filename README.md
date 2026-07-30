# wsl-extended

WSL distrolarını ve `wslc` kapsayıcılarını tek bir terminal arayüzünden yönetmek için Go ile yazılmış TUI.

Sekmeli yapı: **Distros · Containers · Images · Volumes · Networks**

## Durum

**Faz 1 tamam — salt okunur.** Uygulama listeleri gerçek veriyle çiziyor, sekmeler
arasında gezinilebiliyor ve 2 saniyede bir etkin sekmeyi yeniliyor. Durum değiştiren
komutlar (start/stop/remove) henüz yok; onay diyaloğuyla birlikte Faz 2'de gelecek.

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
| `r` | Yenile |
| `?` | Yardımı genişlet |
| `q` | Çık |

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

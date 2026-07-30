# wsl-extended — Tasarım Planı

## 1. Alınan kararlar

| Konu | Karar |
|---|---|
| Kapsam | `wsl.exe` + `wslc.exe` birlikte, sekmeli tek uygulama |
| Kütüphane | Bubble Tea + Lipgloss + bubbles |
| Yıkıcı işlemler | y/n onayı; geri alınamazlarda ad yazdırma |
| İlk sürüm özellikleri | Canlı log + stats, tek tuşla exec/shell, distro export/import, konfig editörü |

## 2. Ortam tespitleri (bu makinede doğrulandı)

- WSL sürümü 2.9.4.0, çekirdek 6.18.35.2-1, tek distro: `FedoraLinux-44` (Stopped)
- `wslc.exe` yolu: `C:\Program Files\WSL\wslc.exe`
- `wslc` komut yüzeyi Docker'a çok yakın: `container`, `image`, `network`, `volume`,
  `registry`, `system`, `attach`, `build`, `create`, `exec`, `export`, `import`,
  `logs`, `stats`, `run`, `start`, `stop`, `kill`, `remove`, `rmi`, `inspect`, `tag`

### Kritik bulgu 1 — `wsl.exe` UTF-16LE yazar

`wsl.exe --list --verbose` çıktısının ham byte'ları `4E 00 41 00 4D 00 45 00` şeklinde,
yani UTF-16LE. Go'da doğrudan okunursa her karakter arasında NUL byte gelir ve parse çöker.

**Çözüm:** her `wsl.exe` çağrısına `WSL_UTF8=1` ortam değişkeni verilir. Doğrulandı:
bu değişkenle çıktı temiz UTF-8 olur. Bu, `internal/wsl` paketindeki tek bir
komut kurucu fonksiyonda merkezîleştirilir; hiçbir yerde çıplak `exec.Command("wsl", ...)` çağrılmaz.

### Kritik bulgu 2 — `wslc` JSON verir, tablo parse etmeye gerek yok

`wslc list --format json`, `wslc images --format json` destekleniyor
(varsayılan `table`). Kırılgan sütun parse'ı yerine doğrudan `encoding/json` kullanılacak.
`wsl.exe` tarafında JSON yok; orada sütun parse'ı kaçınılmaz.

### Kritik bulgu 3 — çıktı yerelleştirilmiş

Bu makinede `wslc --help` Türkçe geliyor. Yani **hata mesajlarına ve durum metinlerine
göre dallanma yapılamaz**; dil değişince kırılır. Karar verirken yalnızca çıkış kodu (exit code)
ve JSON alanları kullanılacak. `wsl --list --verbose` içindeki `STATE` sütunu (`Running`/`Stopped`)
İngilizce kalıyor gibi görünüyor, ama yine de savunmacı parse edilecek: eşleşmezse `Unknown`.

## 3. Mimari

```
cmd/wsl-extended/main.go      — giriş noktası, tea.NewProgram
internal/wsl/                 — wsl.exe sarmalayıcı (WSL_UTF8=1, tablo parse)
internal/wslc/                — wslc.exe sarmalayıcı (--format json)
internal/ui/                  — Bubble Tea modelleri
  app.go                      — kök model, sekme yönlendirme, global tuşlar
  tabs/distros.go
  tabs/containers.go
  tabs/images.go
  tabs/volumes.go
  tabs/networks.go
  components/confirm.go       — onay diyaloğu (y/n + ad yazdırma modu)
  components/logview.go       — canlı log akışı (viewport)
  components/statusbar.go
  theme/                      — Lipgloss stilleri, tek yerden renk
```

**Katman kuralı:** `internal/wsl` ve `internal/wslc` Bubble Tea'yi bilmez; saf Go
tipleri ve `error` döndürür. UI katmanı bunları `tea.Cmd` içine sarar. Böylece
CLI sarmalayıcıları TUI olmadan test edilebilir.

### Async model

Uzun süren işler (`wsl --export` bir distroda dakikalar sürebilir) asla `Update` içinde
senkron çalıştırılmaz — arayüz donar. Her iş `tea.Cmd` olarak başlatılır, bitince
mesaj olarak geri döner. Canlı log/stats için komut `StdoutPipe` ile açılır ve
satırlar bir kanaldan `tea.Msg` olarak akıtılır.

### Yenileme

Poll aralığı 2 sn (`tea.Tick`), ama yalnızca aktif sekme için. Manuel yenileme `r`.
Bir işlem çalışırken poll duraklatılır ki liste altından kaymasın.

## 4. Tuş haritası (taslak)

| Tuş | İşlev |
|---|---|
| `Tab` / `Shift+Tab` | Sekme değiştir |
| `1..5` | Doğrudan sekmeye git |
| `j/k`, ok tuşları | Satır gezinme |
| `Enter` | Shell/exec (distroya `wsl -d`, kapsayıcıya `wslc exec -it`) |
| `l` | Canlı log paneli |
| `s` | Başlat (start) |
| `S` | Durdur (stop) |
| `d` | Sil — onay ister |
| `e` | Export (distro yedeği) |
| `i` | Inspect / detay paneli |
| `c` | Konfig editörü (.wslconfig / wsl.conf) |
| `/` | Filtrele |
| `r` | Yenile |
| `?` | Yardım |
| `q` | Çık |

## 5. Fazlar

**Faz 1 — İskelet ve okuma**
Sekme çatısı, distro listesi (`wsl -l -v`), kapsayıcı/imaj listesi (JSON), durum çubuğu, tema.
Hiçbir yazma işlemi yok. Bu faz sonunda uygulama gerçek veriyle çalışır hâlde.

**Faz 2 — Temel aksiyonlar**
start/stop/terminate, container start/stop/kill/remove, onay diyaloğu bileşeni.
Ad yazdırma modu burada devreye girer.

**Faz 3 — Öldürücü özellikler**
Canlı log akışı, `stats` tablosu, `Enter` ile shell'e düşüp geri dönme
(`tea.ExecProcess` ile — terminal kontrolü devredilir, çıkınca TUI restore edilir).

**Faz 4 — Distro yedekleme**
export/import/klonla, ilerleme göstergesi, hedef yol seçici.
Export sırasında distro otomatik durdurulmalı (tutarlı yedek için) — kullanıcıya sorulur.

**Faz 5 — Konfig editörü**
`.wslconfig` (Windows tarafı, `%UserProfile%\.wslconfig`) ve distro içi `/etc/wsl.conf`.
Form ile düzenle, **yazmadan önce diff göster**. Yedek kopya al (`.wslconfig.bak`).
`.wslconfig` değişikliği `wsl --shutdown` gerektirir — kullanıcı uyarılır.

## 6. Riskler ve dikkat noktaları

- **`wsl --unregister` distroyu kalıcı siler.** Yedeği yoksa geri dönüş yok.
  Ad yazdırma zorunlu; ayrıca "önce export al" kısayolu önerilecek.
- **`tea.ExecProcess` şart.** Shell'e düşerken TUI'nin alternate screen'i bırakması gerekir;
  `exec.Command` doğrudan çalıştırılırsa terminal bozulur.
- **Windows'ta pty yok.** `wslc exec -it` interaktif çalıştırma `tea.ExecProcess` üzerinden
  yapılacak; TUI içi gömülü terminal (pty) hedeflenmiyor — Windows'ta ConPTY işi
  ciddi ölçüde karmaşıklaştırır.
- **Yol boşlukları.** `C:\Program Files\WSL\wslc.exe` boşluk içeriyor;
  komutlar shell string'i olarak değil, argüman dizisi olarak kurulacak.
- **`wslc` yoksa** uygulama çökmemeli; kapsayıcı sekmeleri "wslc bulunamadı" durumu göstermeli.

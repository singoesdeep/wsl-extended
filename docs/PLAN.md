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

> Durum: Faz 1 ve Faz 2 tamamlandı.

**Faz 1 — İskelet ve okuma** ✅
Sekme çatısı, distro listesi (`wsl -l -v`), kapsayıcı/imaj listesi (JSON), durum çubuğu, tema.
Hiçbir yazma işlemi yok. Bu faz sonunda uygulama gerçek veriyle çalışır hâlde.

**Faz 2 — Temel aksiyonlar** ✅
start/stop/terminate, container start/stop/kill/remove, onay diyaloğu bileşeni.
Ad yazdırma modu burada devreye girdi.

Uygulamada eklenen iki güvenlik kuralı:
- Onay açıkken tuşlar arkadaki listeye sızmaz ve otomatik yenileme durur.
  Aksi hâlde onay beklenirken liste tazelenip imleç kayar ve işlem yanlış
  hedefe uygulanabilirdi.
- Ad yazdırma kipinde `y` onay tuşu değildir; yazılan metnin parçasıdır.
  Böylece "y'ye basma" refleksi geri dönüşü olmayan bir işlemi tetikleyemez.

`wsl.exe`'de doğrudan "başlat" komutu olmadığından `Start`, distroda hemen çıkan
bir komut çalıştırır: `wsl -d <ad> --exec /bin/sh -c "exit 0"`. Gerçek distroda
doğrulandı (bkz. `lifecycle_integration_test.go`).

**Faz 3 — Öldürücü özellikler** ✅
Canlı log akışı, `stats` tablosu, `Enter` ile shell'e düşüp geri dönme
(`tea.ExecProcess` ile — terminal kontrolü devredilir, çıkınca TUI restore edilir).

Uygulamada netleşenler:
- `wslc stats` Docker'ın aksine **akış yapmaz**, anlık görüntü döndürür ve
  `--format json` destekler. Bu yüzden panel açıkken düzenli aralıkla yeniden
  çağrılıyor; ayrı bir akış yönetimine gerek kalmadı.
- Günlük akışında kapsayıcının stderr çıktısı da stdout'a katılıyor; aksi hâlde
  stderr'e yazan uygulamaların günlüklerinin yarısı görünmezdi.
- Akış kanalı `ctx` iptalinde kapanıyor. Panel kapatılırken `cancel()`
  çağrılmazsa hem wslc süreci hem de okuyan goroutine arkada kalırdı;
  `TestStreamLinesStopsOnCancel` tam olarak bunu bekliyor.
- Günlükler 2000 satırlık halka tamponda tutuluyor, uzun açık kalan panel
  belleği şişirmiyor.

Doğrulama boşluğu: bu makinede hiç kapsayıcı olmadığı için `logs -f` ve
`exec -it` gerçek veriyle denenemedi.

**Faz 4 — Distro yedekleme** ✅
export/import/klonla, ilerleme göstergesi, hedef yol seçici.
Export sırasında distro otomatik durdurulur (tutarlı yedek için) — onay ekranı söyler.

Uygulamada netleşenler:
- `wsl --export` yüzde bildirmiyor. İlerleme, yazılmakta olan dosyanın o ana
  kadarki boyutundan okunuyor: dürüst ve ek süreç gerektirmiyor.
- Arşiv biçimi uzantıdan türetiliyor (`.tar`, `.tar.gz`, `.vhdx`). Biçim yanlış
  seçilirse dosya yazılıyor ama içeri aktarılamıyor, bu yüzden `--format` ve
  `--vhd` bayrakları otomatik ekleniyor.
- Klonlama ayrı bir komut olarak eklenmedi: export + farklı adla import zaten
  klonlamadır ve import formu bunu anlatıyor. Ayrı bir tuş, aynı işin ikinci
  bir kod yolu olurdu.
- `unregister` onayına yedek alma kısayolu **bilerek** bağlanmadı. Ad yazdırma
  kipinde her tuş yazılan metnin parçasıdır; oraya kısayol koymak kipin
  bütünlüğünü bozardı. Onun yerine diyalog, esc ile çıkıp `e` ile yedek almayı
  öneriyor.

Gerçek WSL'de uçtan uca doğrulandı: `FedoraLinux-44` dışa aktarıldı (579 MB,
7 sn), geçici bir distro olarak içeri aktarıldı, sonra kaydından düşürüldü.
Makine başlangıçtaki durumuna döndü.

**Faz 5 — Konfig editörü** ✅
`.wslconfig` (Windows tarafı, `%UserProfile%\.wslconfig`) ve distro içi `/etc/wsl.conf`.
Form ile düzenle, **yazmadan önce diff göster**. Yedek kopya al (`.wslconfig.bak`).
`.wslconfig` değişikliği `wsl --shutdown` gerektirir — kullanıcı uyarılır.

Uygulamada netleşenler:
- Düzenleme, dosyayı ayrıştırıp yeniden üreterek değil, **satırlar üzerinde
  yerinde** yapılıyor. Yeniden üretme yaklaşımı kullanıcının yorumlarını,
  boş satırlarını ve bu araçta karşılığı olmayan anahtarlarını sessizce
  silerdi. Fedora'nın gerçek `wsl.conf` dosyasında da yorum satırları var.
- Boş bırakılan alan, dosyaya boş değer yazmak yerine anahtarı **siliyor**;
  böylece WSL kendi varsayılanına dönüyor.
- `.wslconfig` UTF-8 BOM ile kaydedilmiş olabilir (Windows editörleri). BOM
  temizlenmezse ilk bölüm başlığı tanınmaz ve dosyadaki tüm ayarlar görünmez
  olurdu; `Parse` bunu kırpıyor.
- CRLF satır sonu kullanan dosyalar kaydedilirken biçimini koruyor.
- `wsl.conf` okuma/yazma distro içinde root olarak yapılıyor. Dosyanın var olup
  olmadığı ayrımı kabuk tarafında (`2>/dev/null || true`) çözüldü, çünkü `cat`
  hata mesajı distronun diline göre değişir.

Doğrulama: `ReadConf` gerçek distroda çalıştırıldı ve Fedora'nın `wsl.conf`
içeriği (yorumlar + `[boot] systemd=true`) doğru okundu. Root olarak stdin ile
yazma mekanizması `/tmp` üzerinde sınandı; `/etc/wsl.conf`'a yazma, kullanıcının
distrosunu değiştirmemek için gerçek dosyada denenmedi.

## 5b. Planın ötesi — ikinci tur

Beş faz bittikten sonra eklenenler.

**Görsel dil: minimal ve tipografik.** Çerçeve ve dolu arka plan kaldırıldı;
hiyerarşi boşluk, hizalama, kalınlık ve renk tonuyla kuruluyor. Seçili satır
dolu bir bloğa dönüşmediği için durum renkleri seçiliyken de okunur kalıyor —
eski tasarımda seçili satırda hücre renkleri kayboluyordu.

**`s` tek tuşa indi.** Duruma göre davranıyor: çalışanı durduruyor, durmuşu
başlatıyor. Ayrı `S` tuşu kaldırıldı.

**Görünen ad (takma ad).** WSL'de yeniden adlandırma komutu yok. Registry'deki
`DistributionName` değiştirilebilir ama hata hâlinde distro görünmez olur.
Bunun yerine ad, uygulamanın kendi verisinde saklanıyor
(`%LOCALAPPDATA%\wsl-extended\data.json`); komutlar her zaman gerçek adı
hedefliyor ve eşleme durum çubuğunda görünüyor. Registry'ye hiç dokunulmuyor.

**Store sekmesi.** `wsl --list --online` kataloğu. Kurulum `--no-launch` ile
yapılıyor: bu bayrak olmadan wsl.exe kurulumun ardından dağıtımı açıp kullanıcı
hesabı sormaya çalışır ve TUI kilitlenir. Kurulum çıktısı canlı akıyor; wsl.exe
ilerlemeyi taşıma dönüşüyle (`\r`) güncellediği için okuyucu hem `\n` hem `\r`
ile bölüyor — yalnızca `\n` arayan bir okuyucu kurulum bitene kadar tek satır
göstermezdi.

Katalog ayrıştırması biçime dayanıyor: geçerli satırın ilk alanı yalnızca harf,
rakam, nokta, tire ve alt çizgi içerir. Başlıktaki ve açıklamalardaki
yerelleştirilmiş metinler böyle eleniyor.

**Detay paneli (`I`).** Statik bilgiler registry'den (`Lxss` altındaki GUID,
`BasePath`, `DefaultUid`) ve dosya sisteminden (`ext4.vhdx` boyutu) okunuyor;
`golang.org/x/sys/windows/registry` kullanılıyor. Canlı bilgiler (çekirdek, IP,
disk kullanımı) yalnızca distro **zaten çalışıyorsa** toplanıyor — bilgi almak
için distro başlatmak istenmeyen bir yan etki olurdu. Canlı veriler tek bir
kabuk çağrısında alınıyor.

**Disk işlemleri menüsü (`D`).** `wsl --manage` altındaki resize / set-sparse /
move işlemleri tek menüde. Resize ve move distro kapalıyken çalıştığı için menü
distro çalışıyorken uyarı gösteriyor.

Doğrulama: `Describe` gerçek makinede çalıştırıldı (GUID, kurulum dizini,
1.07 GB disk boyutu doğru okundu, distro başlatılmadı) ve `ListOnline` 22
dağıtımı açıklama satırlarını eleyerek ayrıştırdı. Kurulum akışı gerçek bir
indirmeyle denenmedi.

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

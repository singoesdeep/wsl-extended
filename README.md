# wsl-extended

WSL distrolarını ve `wslc` kapsayıcılarını tek bir terminal arayüzünden yöneten,
Go ile yazılmış bir TUI.

![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white)
![Platform](https://img.shields.io/badge/platform-Windows-0078D6?logo=windows&logoColor=white)
![License](https://img.shields.io/badge/license-MIT-green)

```
   1 Distros (3)    2 Store    3 Containers    4 Images    5 Volumes    6 Networks
────────────────────────────────────────────────────────────────────────────────────
   NAME                                        STATE         VERSION  DEFAULT
 ▍● FedoraLinux-44                             Running       2        ✓
   ○ openSUSE-Tumbleweed                       Stopped       2
   ○ kali-linux                                Stopped       2

 tab sekme  ·  j/k gezin  ·  / ara  ·  s başlat/durdur  ·  enter kabuk  ·  ? tümü
 wsl-extended  ·  WSL 2.9.4.0
```

## Ne yapar

`wsl.exe` ve `wslc.exe` komutlarını ezberlemeden WSL'i yönetmeni sağlar:

- **Distroları** başlat, durdur, sil, varsayılan yap, kabuğa gir
- **Yedekle ve geri yükle** — tek tuşla arşive aktar, arşivden yeni distro oluştur
- **Yeni dağıtım kur** — `wsl --list --online` kataloğundan seç, indir
- **Yapılandırmayı düzenle** — `.wslconfig` ve `wsl.conf` için form, kaydetmeden önce fark göster
- **Diski yönet** — büyüt, seyrek disk aç/kapat, başka konuma taşı
- **Kapsayıcıları yönet** — liste, canlı günlük, kaynak kullanımı, kabuk, imaj çekme
- **Ara ve toplu işlem yap** — `/` ile süz, `space` ile işaretle

## Gereksinimler

- Windows 10/11
- Go 1.26 veya üstü (derlemek için)
- WSL 2.9+ (`wsl.exe`)
- `wslc.exe` — yalnızca kapsayıcı sekmeleri için. Yoksa uygulama çalışır, o sekmeler bilgi ekranı gösterir.

## Kurulum

```bash
git clone https://github.com/singoesdeep/wsl-extended.git
```

```bash
cd wsl-extended && go build -o bin/wsl-extended.exe ./cmd/wsl-extended
```

Ya da doğrudan çalıştır:

```bash
go run ./cmd/wsl-extended
```

## Tuşlar

Tüm kısayollar küçük harftir. Yardım satırı bulunduğun sekmeye göre değişir ve
terminale sığmazsa kırpılır; `?` tam listeyi açar.

| Tuş | İşlev |
|---|---|
| `tab` / `shift+tab` | Sekme değiştir |
| `1`–`6` | Doğrudan sekmeye git |
| `j` / `k`, ok tuşları | Satır gezinme |
| `home` / `end` | Listenin başı / sonu |
| `/` | Ara / süz |
| `space` | Satırı işaretle (toplu işlem) |
| `esc` | Süzgeci ve işaretleri temizle |
| `s` | Başlat veya durdur — duruma göre |
| `enter` | Kabuğa gir; Store sekmesinde kurulumu başlatır |
| `v` | Distro detay paneli |
| `o` | Şununla aç (Gezgin / VS Code / yeni pencere) |
| `m` | Disk işlemleri (büyüt / seyrek / taşı) |
| `n` | Görünen adı değiştir |
| `e` | Dışa aktar (yedekle) |
| `i` | Arşivden distro oluştur |
| `c` | Ayarlar (`.wslconfig`, `wsl.conf`, WSL'i kapat, güncelle) |
| `w` | Sistem durumu |
| `l` | Canlı günlük (kapsayıcı) |
| `t` | Kaynak kullanımı |
| `p` | İmaj çek / kapsayıcı çalıştır |
| `x` | Kapsayıcıyı sonlandır |
| `u` | Varsayılan distro yap |
| `d` | Sil |
| `r` | Yenile |
| `q` | Çık |

## Güvenlik yaklaşımı

Durum değiştiren her işlem onaydan geçer ve iki kip vardır:

- **y/n** — geri alınabilir işlemler: başlat, durdur, kill, kapsayıcı/imaj/ağ silme.
- **Adı yazma** — geri dönüşü olmayanlar: `wsl --unregister` ve birim silme. Hedefin
  adı harfi harfine yazılmadan `enter` çalışmaz, ve bu kipte `y` bir onay tuşu değil,
  yazılan metnin parçasıdır.

Onay açıkken tuşlar arkadaki listeye ulaşmaz ve otomatik yenileme durur; böylece onay
beklerken imleç kayıp işlem yanlış hedefe uygulanamaz.

**Distro ve birim silme toplu yapılamaz.** Bu ikisinin koruması adı yazdırmaktır ve tek
onayla birden çok distro silmek o korumayı anlamsız kılardı.

## Öne çıkan davranışlar

**Görünen ad.** WSL'de yeniden adlandırma komutu yoktur; registry'deki
`DistributionName` değiştirilebilir ama hata hâlinde distro görünmez olur. Bunun yerine
`n` ile verdiğin ad uygulamanın kendi verisinde (`%LOCALAPPDATA%\wsl-extended\data.json`)
saklanır — komutlar her zaman gerçek adla çalışır ve eşleme durum çubuğunda görünür.

**Yedekleme.** `e` distroyu arşive yazar; biçim uzantıdan belirlenir (`.tar`, `.tar.gz`,
`.vhdx`). Distro çalışıyorsa tutarlı arşiv için önce durdurulur. `wsl --export` yüzde
bildirmediğinden ilerleme, yazılmakta olan dosyanın boyutundan okunur.

**Yapılandırma düzenleme.** Düzenleme dosyanın satırları üzerinde yerinde yapılır:
yorumların, boş satırların ve bu araçta karşılığı olmayan anahtarların hiçbiri kaybolmaz.
Kaydetmeden önce fark gösterilir, eski hâl `.bak` olarak saklanır. Boş bırakılan alan
anahtarı siler, yani WSL kendi varsayılanına döner.

**Dağıtım kurma.** `--no-launch` ile kurulur; bu bayrak olmadan `wsl.exe` kurulum biter
bitmez dağıtımı açıp hesap sormaya çalışır ve arayüzü kilitler. Kurulum sırasında terminal
`wsl.exe`'ye devredilir, böylece kendi indirme yüzdesini çizebilir.

**Detay paneli.** `v` distro kapalıysa onu **başlatmaz**; registry ve dosya sisteminden
okunabilenleri gösterir. Çalışıyorsa çekirdek, IP ve disk kullanımı eklenir.

## Geliştirme

```bash
go test ./...
```

Gerçek `wsl.exe` ve `wslc.exe` çağıran entegrasyon testleri ayrı etikettedir. Yalnızca
okuma ve geri alınabilir başlat/durdur yaparlar; silme komutları hiçbir testte çağrılmaz:

```bash
go test -tags integration ./...
```

### Mimari

```
cmd/wsl-extended/     giriş noktası
internal/wsl/         wsl.exe sarmalayıcı
internal/wslc/        wslc.exe sarmalayıcı
internal/wslconf/     .wslconfig / wsl.conf düzenleme
internal/store/       uygulama verisi (takma adlar)
internal/ui/          Bubble Tea katmanı
```

`internal/wsl`, `internal/wslc` ve `internal/wslconf` arayüz katmanını bilmez; saf Go
tipleri ve `error` döndürür, böylece TUI olmadan test edilebilirler.

İki tuzak koda gömülü olarak çözülmüştür: `wsl.exe` çıktısını UTF-16LE yazdığı için her
çağrıya `WSL_UTF8=1` geçilir, ve CLI çıktıları yerelleştirilmiş olabildiğinden hiçbir
karar hata metnine bakarak verilmez — yalnızca çıkış kodu ve JSON alanları kullanılır.

## Bilinen sınırlar

- Yalnızca Windows'ta çalışır (`wsl.exe` ve registry'ye bağımlıdır).
- `wslc` JSON şeması belgelenmemiştir; alan adları Docker çıktısı örnek alınarak
  yazılmış ve tip sürprizlerine karşı savunmacı tutulmuştur.
- Kapsayıcı günlüğü/exec ve `/etc/wsl.conf` yazma henüz gerçek veriyle sınanmadı.
- TUI içine gömülü terminal (pty) yoktur; kabuk açarken terminal komuta devredilir.

## Lisans

[MIT](LICENSE)

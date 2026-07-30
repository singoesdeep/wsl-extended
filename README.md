# wsl-extended

WSL distrolarını ve `wslc` kapsayıcılarını tek bir terminal arayüzünden yönetmek için Go ile yazılmış TUI.

Sekmeli yapı: **Distros · Store · Containers · Images · Volumes · Networks**

## Durum

**Planlanan beş fazın tamamı bitti.** Listeler, aksiyonlar (onay kapısıyla), canlı
günlük akışı, kaynak kullanımı paneli, tek tuşla kabuk, distro yedekleme
(export/import) ve yapılandırma düzenleyicisi.

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
| `1`–`6` | Doğrudan sekmeye git |
| `j` / `k`, ok tuşları | Satır gezinme |
| `g` / `G` | Listenin başı / sonu |
| `s` | **Başlat veya durdur** — çalışıyorsa durdurur, duruyorsa başlatır |
| `Enter` | Kabuğa gir; Store sekmesinde kurulumu başlatır |
| `I` | Distro detay paneli |
| `D` | Disk işlemleri menüsü (büyüt / seyrek / taşı) |
| `n` | Görünen adı değiştir (takma ad) |
| `e` | Distroyu dışa aktar (yedekle) |
| `i` | Arşivden distro oluştur |
| `c` | `.wslconfig` düzenleyicisi (tüm WSL) |
| `C` | Seçili distronun `wsl.conf` düzenleyicisi |
| `L` | Canlı günlük paneli (kapsayıcı) |
| `t` | Kaynak kullanımı paneli |
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

## Görünen ad (takma ad)

`n` ile bir distroya istediğin adı verebilirsin. **WSL'de yeniden adlandırma
komutu yoktur**; registry'deki `DistributionName` değerini değiştirmek mümkündür ama
yanlış giderse distro görünmez hâle gelir. Bu yüzden ad, uygulamanın kendi verisinde
(`%LOCALAPPDATA%\wsl-extended\data.json`) saklanır: listede senin verdiğin ad görünür,
tüm komutlar gerçek adla çalışır ve eşleme durum çubuğunda gösterilir. Alanı boş
bırakırsan gerçek ada döner.

## Store — dağıtım kurma

`2` sekmesi `wsl --list --online` kataloğunu gösterir, kurulu olanlar işaretlenir.
`Enter` seçili dağıtımı indirip kurar; ilerleme canlı akar.

Kurulum `--no-launch` ile yapılır: dağıtım kurulumdan sonra otomatik açılmaz.
Kullanıcı hesabını, ilk kez `Enter` ile kabuğa girdiğinde oluşturursun. (Bu bayrak
olmadan `wsl.exe` kurulum biter bitmez dağıtımı açıp hesap sormaya çalışır ve
arayüzü kilitler.)

## Detay paneli

`I` seçili distronun her şeyini tek ekranda gösterir: durum, WSL sürümü, registry
kimliği, kurulum dizini, disk dosyası ve boyutu, varsayılan UID. Distro çalışıyorsa
çekirdek sürümü, IP adresi ve kök disk kullanımı da eklenir.

Distro **kapalıysa panel onu başlatmaz** — yalnızca registry ve dosya sisteminden
okunabilen bilgileri gösterir ve canlı veri için `s` ile başlatmanı önerir. Bilgi
almak uğruna distro başlatmak, istemediğin bir yan etki olurdu.

## Disk işlemleri

`D` menüsü `wsl --manage` altındaki işlemleri toplar: diski büyüt (`--resize`),
seyrek diski aç/kapat (`--set-sparse`), başka konuma taşı (`--move`). Seyrek disk
açıkken silinen dosyaların yeri Windows tarafında otomatik geri kazanılır.

Büyütme ve taşıma distro **kapalıyken** çalışır; distro çalışıyorsa menü bunu
önceden söyler.

## Yedekleme

`e` seçili distroyu tek bir arşive yazar. Biçim uzantıdan belirlenir: `.tar`,
`.tar.gz` ya da `.vhdx`. Distro çalışıyorsa tutarlı bir arşiv için önce durdurulur —
onay ekranı bunu söyler. Arşiv yazılırken durum çubuğu o ana kadar yazılan boyutu
gösterir.

`i` bir arşivden yeni distro oluşturur. **Klonlamak** için önce `e` ile yedek al,
sonra `i` ile başka bir ad ver.

## Yapılandırma düzenleyicisi

`c` Windows tarafındaki `%UserProfile%\.wslconfig` dosyasını (bellek, işlemci, takas,
ağ kipi…), `C` ise seçili distronun içindeki `/etc/wsl.conf` dosyasını (systemd,
varsayılan kullanıcı, automount, interop…) düzenler.

Alanlar arasında `j`/`k` ile gezinir, `enter` ile düzenler, `backspace` ile
temizlersin. Boş bırakılan alan dosyadan **silinir** — böylece WSL kendi
varsayılanına döner. `s` kaydeder, ama önce yazılacak farkı gösterip onay ister.

Düzenleme dosyanın satırları üzerinde yapılır: yorumların, boş satırların ve bu
araçta karşılığı olmayan anahtarların hiçbiri kaybolmaz. Kaydetmeden önce eski
hâl `.bak` uzantısıyla saklanır.

`.wslconfig` değişikliği ancak WSL sanal makinesi yeniden kurulduğunda etkili olur;
kaydettikten sonra `X` ile WSL'i kapatman gerekir. `wsl.conf` için ilgili distroyu
yeniden başlatmak yeterlidir. `C` tuşu dosyayı okumak için distroyu başlatır.

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

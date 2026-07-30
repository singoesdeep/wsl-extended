// Package wslc, wslc.exe (WSL Kapsayıcı CLI) aracını sarmalar.
//
// wsl.exe'nin aksine wslc `--format json` destekler, bu yüzden burada tablo
// ayrıştırma yoktur. Ancak JSON şeması belgelenmemiştir; alan adları Docker'ın
// `--format json` çıktısı örnek alınarak yazılmış ve eksik/farklı tipli
// alanların ayrıştırmayı bozmaması için savunmacı tutulmuştur.
package wslc

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

// fallbackPath, wslc PATH üzerinde bulunamazsa denenecek varsayılan konum.
const fallbackPath = `C:\Program Files\WSL\wslc.exe`

// Available, wslc.exe'nin çalıştırılabilir olup olmadığını söyler. Kapsayıcı
// sekmeleri, wslc kurulu değilken uygulamayı çökertmek yerine bilgi gösterir.
func Available() bool {
	_, err := binary()
	return err == nil
}

func binary() (string, error) {
	if p, err := exec.LookPath("wslc.exe"); err == nil {
		return p, nil
	}
	if _, err := exec.LookPath(fallbackPath); err == nil {
		return fallbackPath, nil
	}
	return "", errors.New("wslc.exe bulunamadı")
}

func runJSON(ctx context.Context, v any, args ...string) error {
	bin, err := binary()
	if err != nil {
		return err
	}

	out, err := exec.CommandContext(ctx, bin, args...).Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && len(ee.Stderr) > 0 {
			return fmt.Errorf("wslc %s: %s", strings.Join(args, " "),
				strings.TrimSpace(string(ee.Stderr)))
		}
		return fmt.Errorf("wslc %s: %w", strings.Join(args, " "), err)
	}

	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return nil // çıktı yok: boş liste
	}
	if err := json.Unmarshal([]byte(trimmed), v); err != nil {
		return fmt.Errorf("wslc %s: JSON ayrıştırılamadı: %w", strings.Join(args, " "), err)
	}
	return nil
}

// Text, JSON'da string, sayı ya da string dizisi olarak gelebilen alanları
// karşılayan esnek bir tiptir. Şema belgelenmediği için tip sürprizlerinin tüm
// listeyi düşürmesini engeller.
type Text string

func (t Text) String() string { return string(t) }

func (t *Text) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		*t = Text(s)
		return nil
	}
	var ss []string
	if err := json.Unmarshal(b, &ss); err == nil {
		*t = Text(strings.Join(ss, ", "))
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(b, &n); err == nil {
		*t = Text(n.String())
		return nil
	}
	// Beklenmeyen tip: alanı boş bırak, kaydın geri kalanını koru.
	*t = ""
	return nil
}

type Container struct {
	ID        Text `json:"ID"`
	Names     Text `json:"Names"`
	Image     Text `json:"Image"`
	State     Text `json:"State"`
	Status    Text `json:"Status"`
	Ports     Text `json:"Ports"`
	CreatedAt Text `json:"CreatedAt"`
}

// Name, listede gösterilecek adı verir; ad yoksa kimliğe düşer.
func (c Container) Name() string {
	if n := strings.TrimPrefix(c.Names.String(), "/"); n != "" {
		return n
	}
	return c.ID.String()
}

func (c Container) IsRunning() bool {
	return strings.EqualFold(c.State.String(), "running")
}

type Image struct {
	ID         Text `json:"ID"`
	Repository Text `json:"Repository"`
	Tag        Text `json:"Tag"`
	Size       Text `json:"Size"`
	CreatedAt  Text `json:"CreatedAt"`
}

type Volume struct {
	Name       Text `json:"Name"`
	Driver     Text `json:"Driver"`
	Mountpoint Text `json:"Mountpoint"`
	Scope      Text `json:"Scope"`
}

type Network struct {
	ID     Text `json:"ID"`
	Name   Text `json:"Name"`
	Driver Text `json:"Driver"`
	Scope  Text `json:"Scope"`
}

// Stat, tek bir kapsayıcının kaynak kullanımı anlık görüntüsüdür.
//
// `wslc stats` Docker'ın aksine akış yapmaz; her çağrı tek bir görüntü döndürür,
// bu yüzden arayüz tarafında düzenli aralıkla yeniden çağrılır.
type Stat struct {
	ID       Text `json:"ID"`
	Name     Text `json:"Name"`
	CPUPerc  Text `json:"CPUPerc"`
	MemUsage Text `json:"MemUsage"`
	MemPerc  Text `json:"MemPerc"`
	NetIO    Text `json:"NetIO"`
	BlockIO  Text `json:"BlockIO"`
	PIDs     Text `json:"PIDs"`
}

func Stats(ctx context.Context) ([]Stat, error) {
	var ss []Stat
	err := runJSON(ctx, &ss, "stats", "--all", "--format", "json")
	return ss, err
}

// StreamLogs, kapsayıcının günlüklerini takip eden bir satır kanalı döndürür.
//
// Kanal, ctx iptal edildiğinde ya da komut sona erdiğinde kapanır. Çağıran
// taraf kanalı sonuna kadar tüketmeli ya da ctx'i iptal etmelidir; aksi hâlde
// okuyan goroutine sızar.
func StreamLogs(ctx context.Context, id string, tail int) (<-chan string, error) {
	bin, err := binary()
	if err != nil {
		return nil, err
	}

	args := []string{"logs", "--follow", "--tail", strconv.Itoa(tail), id}
	cmd := exec.CommandContext(ctx, bin, args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	// Kapsayıcılar günlüklerini sıklıkla stderr'e yazar; ikisi de aynı akışa
	// katılmazsa günlüklerin yarısı kaybolur.
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	ch := make(chan string, 256)
	go func() {
		streamLines(ctx, stdout, ch)
		_ = cmd.Wait()
	}()
	return ch, nil
}

// streamLines, r'deki satırları ch'ye aktarır ve işi bitince ch'yi kapatır.
// ctx iptal edilirse yazmayı bırakır, böylece kimse okumadığında bloke olmaz.
func streamLines(ctx context.Context, r io.Reader, ch chan<- string) {
	defer close(ch)

	sc := bufio.NewScanner(r)
	// Varsayılan 64 KiB sınırı uzun günlük satırlarında taşar.
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for sc.Scan() {
		select {
		case ch <- sc.Text():
		case <-ctx.Done():
			return
		}
	}
}

// PullCommand, imaj indirme komutunu hazırlar. Komut çalıştırılmaz: indirme
// ilerlemesini gösterebilmesi için terminali devralması gerekir.
func PullCommand(image string) (*exec.Cmd, error) {
	bin, err := binary()
	if err != nil {
		return nil, err
	}
	return exec.Command(bin, "pull", image), nil
}

// RunOptions, yeni bir kapsayıcı için verilen seçenekler. Boş alanlar komuta
// eklenmez.
type RunOptions struct {
	Image   string
	Name    string
	Port    string // "8080:80"
	Volume  string // "data:/veri"
	Command string
}

// RunArgs, seçeneklerden `wslc run` argümanlarını üretir.
func (o RunOptions) RunArgs() []string {
	args := []string{"run", "--detach"}
	if o.Name != "" {
		args = append(args, "--name", o.Name)
	}
	if o.Port != "" {
		args = append(args, "--publish", o.Port)
	}
	if o.Volume != "" {
		args = append(args, "--volume", o.Volume)
	}
	args = append(args, o.Image)
	if o.Command != "" {
		args = append(args, strings.Fields(o.Command)...)
	}
	return args
}

// Run, kapsayıcıyı arka planda başlatır.
func Run(ctx context.Context, o RunOptions) error {
	return run(ctx, o.RunArgs()...)
}

// ShellCommand, kapsayıcıda etkileşimli bir kabuk açan komutu hazırlar.
// Komut çalıştırılmaz; terminali devralması için çağırana verilir.
func ShellCommand(id string) (*exec.Cmd, error) {
	bin, err := binary()
	if err != nil {
		return nil, err
	}
	return exec.Command(bin, "exec", "-i", "-t", id, "/bin/sh"), nil
}

// run, çıktısı önemsenmeyen durum değiştiren komutları çalıştırır.
func run(ctx context.Context, args ...string) error {
	bin, err := binary()
	if err != nil {
		return err
	}

	if _, err := exec.CommandContext(ctx, bin, args...).Output(); err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && len(ee.Stderr) > 0 {
			return fmt.Errorf("wslc %s: %s", strings.Join(args, " "),
				strings.TrimSpace(string(ee.Stderr)))
		}
		return fmt.Errorf("wslc %s: %w", strings.Join(args, " "), err)
	}
	return nil
}

func StartContainer(ctx context.Context, id string) error {
	return run(ctx, "start", id)
}

func StopContainer(ctx context.Context, id string) error {
	return run(ctx, "stop", id)
}

// KillContainer, kapsayıcıyı beklemeden sonlandırır.
func KillContainer(ctx context.Context, id string) error {
	return run(ctx, "kill", id)
}

func RemoveContainer(ctx context.Context, id string) error {
	return run(ctx, "remove", id)
}

func RemoveImage(ctx context.Context, id string) error {
	return run(ctx, "rmi", id)
}

// RemoveVolume, birimi ve içindeki veriyi siler. Geri dönüşü yoktur.
func RemoveVolume(ctx context.Context, name string) error {
	return run(ctx, "volume", "remove", name)
}

func RemoveNetwork(ctx context.Context, name string) error {
	return run(ctx, "network", "remove", name)
}

func Containers(ctx context.Context) ([]Container, error) {
	var cs []Container
	err := runJSON(ctx, &cs, "list", "--all", "--format", "json")
	return cs, err
}

func Images(ctx context.Context) ([]Image, error) {
	var is []Image
	err := runJSON(ctx, &is, "images", "--format", "json")
	return is, err
}

func Volumes(ctx context.Context) ([]Volume, error) {
	var vs []Volume
	err := runJSON(ctx, &vs, "volume", "list", "--format", "json")
	return vs, err
}

func Networks(ctx context.Context) ([]Network, error) {
	var ns []Network
	err := runJSON(ctx, &ns, "network", "list", "--format", "json")
	return ns, err
}

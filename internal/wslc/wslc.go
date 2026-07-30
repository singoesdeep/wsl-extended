// Package wslc, wslc.exe (WSL Kapsayıcı CLI) aracını sarmalar.
//
// wsl.exe'nin aksine wslc `--format json` destekler, bu yüzden burada tablo
// ayrıştırma yoktur. Ancak JSON şeması belgelenmemiştir; alan adları Docker'ın
// `--format json` çıktısı örnek alınarak yazılmış ve eksik/farklı tipli
// alanların ayrıştırmayı bozmaması için savunmacı tutulmuştur.
package wslc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
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

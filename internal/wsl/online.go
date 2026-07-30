package wsl

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"strings"
)

// OnlineDistro, `wsl --list --online` ile kurulabilen bir dağıtımdır.
type OnlineDistro struct {
	Name     string // kuruluma verilecek ad
	Friendly string // insan okunur ad
}

// ListOnline, kurulabilir dağıtımları döndürür.
func ListOnline(ctx context.Context) ([]OnlineDistro, error) {
	out, err := run(ctx, "--list", "--online")
	if err != nil {
		return nil, err
	}
	return parseOnline(out), nil
}

// parseOnline, katalog tablosunu ayrıştırır.
//
// Çıktının başındaki açıklama satırları ve sütun başlığı yerelleştirilmiştir,
// bu yüzden metne göre elenemezler. Ayrım biçimden yapılır: geçerli bir satırın
// ilk alanı dağıtım kimliğidir ve yalnızca harf, rakam, nokta, tire ve alt
// çizgi içerir; açıklama cümleleri virgül, tırnak gibi karakterler taşır.
func parseOnline(out string) []OnlineDistro {
	var list []OnlineDistro

	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		name := fields[0]
		if !isDistroID(name) {
			continue
		}
		// Sütun başlığı: ilk alan "NAME" olur ve kimlik biçimine uyar.
		if strings.EqualFold(name, "NAME") {
			continue
		}

		list = append(list, OnlineDistro{
			Name:     name,
			Friendly: strings.Join(fields[1:], " "),
		})
	}
	return list
}

func isDistroID(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '-', r == '_', r == '.':
		default:
			return false
		}
	}
	return true
}

// InstallCommand, dağıtım kurulum komutunu hazırlar. Komut çalıştırılmaz;
// çağıran taraf terminali ona devreder.
//
// Kurulum `--no-launch` ile yapılır: bu bayrak olmadan wsl.exe kurulumun
// ardından dağıtımı açıp kullanıcı hesabı sormaya çalışır.
//
// Terminalin devredilmesi şart: wsl.exe indirme yüzdesini yalnızca gerçek bir
// konsola bağlıyken çizer. Çıktı bir boruya yönlendirildiğinde ilerleme
// çubuğunu bastırır ve kullanıcı dakikalarca boş ekrana bakar.
func InstallCommand(name string) *exec.Cmd {
	cmd := exec.Command("wsl.exe", "--install", name, "--no-launch")
	cmd.Env = append(os.Environ(), "WSL_UTF8=1")
	return cmd
}

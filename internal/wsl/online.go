package wsl

import (
	"bufio"
	"bytes"
	"context"
	"io"
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

// StreamInstall, dağıtım kurulumunu başlatır ve çıktısını satır satır akıtır.
//
// Kurulum `--no-launch` ile yapılır: bu bayrak olmadan wsl.exe kurulumun
// ardından dağıtımı açıp kullanıcı hesabı sormaya çalışır ve arayüzü kilitler.
//
// Kanal, kurulum bitince ya da ctx iptal edilince kapanır.
func StreamInstall(ctx context.Context, name string) (<-chan string, error) {
	cmd := command(ctx, "--install", name, "--no-launch")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	ch := make(chan string, 64)
	go func() {
		streamProgress(ctx, stdout, ch)
		_ = cmd.Wait()
	}()
	return ch, nil
}

// streamProgress, çıktıyı satır satır kanala aktarır ve işi bitince kanalı
// kapatır.
func streamProgress(ctx context.Context, r io.Reader, ch chan<- string) {
	defer close(ch)

	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	sc.Split(scanLinesOrCR)

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		select {
		case ch <- line:
		case <-ctx.Done():
			return
		}
	}
}

// scanLinesOrCR, hem \n hem \r ile biten parçaları satır kabul eder.
//
// wsl.exe indirme yüzdesini aynı satırda taşıma dönüşüyle günceller; yalnızca
// \n arayan bir okuyucu, kurulum bitene kadar tek satır bile göstermez.
func scanLinesOrCR(data []byte, atEOF bool) (int, []byte, error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexAny(data, "\r\n"); i >= 0 {
		return i + 1, data[:i], nil
	}
	if atEOF {
		return len(data), data, nil
	}
	return 0, nil, nil
}

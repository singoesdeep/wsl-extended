// Package wsl, wsl.exe komut satırı aracını sarmalar.
//
// Paket bilerek arayüz katmanından bağımsızdır: yalnızca saf Go tipleri ve
// error döndürür, Bubble Tea'yi bilmez.
package wsl

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// State, bir distronun çalışma durumu.
type State string

const (
	StateRunning State = "Running"
	StateStopped State = "Stopped"
	StateUnknown State = "Unknown"
)

// Distro, `wsl --list --verbose` çıktısındaki tek bir satırı temsil eder.
type Distro struct {
	Name    string
	State   State
	Version string
	Default bool
}

func (d Distro) IsRunning() bool { return d.State == StateRunning }

// Available, wsl.exe'nin PATH üzerinde bulunup bulunmadığını söyler.
func Available() bool {
	_, err := exec.LookPath("wsl.exe")
	return err == nil
}

// command, wsl.exe çağrısını UTF-8 çıktı verecek şekilde kurar.
//
// WSL_UTF8 ayarlanmazsa wsl.exe çıktısını UTF-16LE yazar; Go tarafında bu, her
// karakterin arasına NUL byte serpiştirilmiş bir metin olarak okunur ve tüm
// ayrıştırma sessizce bozulur. wsl.exe'ye giden her çağrı buradan geçmelidir.
func command(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "wsl.exe", args...)
	cmd.Env = append(os.Environ(), "WSL_UTF8=1")
	return cmd
}

func run(ctx context.Context, args ...string) (string, error) {
	out, err := command(ctx, args...).Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && len(ee.Stderr) > 0 {
			return "", fmt.Errorf("wsl %s: %s", strings.Join(args, " "),
				strings.TrimSpace(string(ee.Stderr)))
		}
		return "", fmt.Errorf("wsl %s: %w", strings.Join(args, " "), err)
	}
	return string(out), nil
}

// List, kurulu distroları döndürür.
func List(ctx context.Context) ([]Distro, error) {
	out, err := run(ctx, "--list", "--verbose")
	if err != nil {
		return nil, err
	}
	return parseList(out), nil
}

// Version, `wsl --version` çıktısının ilk satırındaki sürüm numarasını döndürür.
func Version(ctx context.Context) (string, error) {
	out, err := run(ctx, "--version")
	if err != nil {
		return "", err
	}
	for _, line := range splitLines(out) {
		// Satır yerelleştirilmiştir ("WSL sürümü: 2.9.4.0"), bu yüzden etikete
		// değil, satırdaki ilk nokta içeren sayısal alana bakılır.
		for _, f := range strings.Fields(line) {
			f = strings.Trim(f, ":")
			if strings.Count(f, ".") >= 2 && strings.ContainsAny(f, "0123456789") {
				return f, nil
			}
		}
	}
	return "", nil
}

// Start, distroyu ayağa kaldırır. wsl.exe'de doğrudan "başlat" komutu yoktur;
// distro, içinde bir komut çalıştırıldığında başlar. Kabuk başlatmamak için
// --exec ile hemen çıkan bir komut kullanılır.
func Start(ctx context.Context, name string) error {
	_, err := run(ctx, "-d", name, "--exec", "/bin/sh", "-c", "exit 0")
	return err
}

// ShellCommand, distroda etkileşimli bir kabuk açan komutu hazırlar. Komut
// çalıştırılmaz; terminali devralması için çağırana verilir.
func ShellCommand(name string) *exec.Cmd {
	cmd := exec.Command("wsl.exe", "-d", name)
	cmd.Env = append(os.Environ(), "WSL_UTF8=1")
	return cmd
}

// Terminate, tek bir distroyu durdurur.
func Terminate(ctx context.Context, name string) error {
	_, err := run(ctx, "--terminate", name)
	return err
}

// Shutdown, tüm distroları ve WSL sanal makinesini kapatır.
func Shutdown(ctx context.Context) error {
	_, err := run(ctx, "--shutdown")
	return err
}

// SetDefault, varsayılan distroyu değiştirir.
func SetDefault(ctx context.Context, name string) error {
	_, err := run(ctx, "--set-default", name)
	return err
}

// Unregister, distroyu kaydından düşürür ve diskteki tüm verisini siler.
// Geri dönüşü yoktur; çağıran taraf önce açık onay almalıdır.
func Unregister(ctx context.Context, name string) error {
	_, err := run(ctx, "--unregister", name)
	return err
}

func splitLines(s string) []string {
	return strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
}

// parseList, `wsl --list --verbose` tablosunu ayrıştırır.
//
// Başlık satırı yerelleştirilmiş olabildiği için metnine göre atlanamaz; onun
// yerine son sütunun sayı olmaması ölçüt alınır. Distro adı boşluk
// içerebileceğinden ad, sondaki iki sütun dışındaki her şeydir.
func parseList(out string) []Distro {
	var ds []Distro
	for _, line := range splitLines(out) {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		def := false
		if strings.HasPrefix(line, "*") {
			def = true
			line = strings.TrimSpace(line[1:])
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		version := fields[len(fields)-1]
		if _, err := strconv.Atoi(version); err != nil {
			continue // başlık satırı ya da beklenmeyen biçim
		}

		ds = append(ds, Distro{
			Name:    strings.Join(fields[:len(fields)-2], " "),
			State:   parseState(fields[len(fields)-2]),
			Version: version,
			Default: def,
		})
	}
	return ds
}

// parseState, bilinen durumları normalize eder; tanımadığını olduğu gibi
// saklar, böylece farklı bir dil ya da yeni bir durum değeri veriyi düşürmez.
func parseState(s string) State {
	switch strings.ToLower(s) {
	case "running":
		return StateRunning
	case "stopped":
		return StateStopped
	case "":
		return StateUnknown
	default:
		return State(s)
	}
}

//go:build integration

// Bu testler gerçek wslc.exe'yi çağırır ve yalnızca okuma yapar; hiçbir
// kapsayıcı, imaj ya da birim oluşturmaz veya silmez.
package wslc

import (
	"context"
	"testing"
	"time"
)

func ctx(t *testing.T) context.Context {
	t.Helper()
	c, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	return c
}

// JSON şeması belgelenmediği için buradaki amaç alan adlarını doğrulamak değil,
// her listeleme komutunun hatasız çalışıp geçerli JSON döndürdüğünü görmek.
func TestListingsAgainstRealWSLC(t *testing.T) {
	if !Available() {
		t.Skip("wslc.exe yok")
	}

	t.Run("containers", func(t *testing.T) {
		cs, err := Containers(ctx(t))
		if err != nil {
			t.Fatalf("Containers: %v", err)
		}
		t.Logf("%d kapsayıcı: %+v", len(cs), cs)
	})

	t.Run("images", func(t *testing.T) {
		is, err := Images(ctx(t))
		if err != nil {
			t.Fatalf("Images: %v", err)
		}
		t.Logf("%d imaj: %+v", len(is), is)
	})

	t.Run("volumes", func(t *testing.T) {
		vs, err := Volumes(ctx(t))
		if err != nil {
			t.Fatalf("Volumes: %v", err)
		}
		t.Logf("%d birim: %+v", len(vs), vs)
	})

	t.Run("networks", func(t *testing.T) {
		ns, err := Networks(ctx(t))
		if err != nil {
			t.Fatalf("Networks: %v", err)
		}
		t.Logf("%d ağ: %+v", len(ns), ns)
	})
}

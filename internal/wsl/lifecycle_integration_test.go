//go:build integration

package wsl

import (
	"context"
	"testing"
	"time"
)

// TestStartThenTerminate, distro yaşam döngüsünü gerçek wsl.exe ile sınar.
//
// Test yalnızca zaten durmuş bir distroya dokunur ve işi bittiğinde onu yine
// durmuş hâlde bırakır; çalışan bir distroyu kesintiye uğratmaz. Silme ya da
// kayıttan düşürme gibi geri dönüşü olmayan komutlar hiçbir testte çağrılmaz.
func TestStartThenTerminate(t *testing.T) {
	if !Available() {
		t.Skip("wsl.exe yok")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	before, err := List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	var target string
	for _, d := range before {
		if d.State == StateStopped {
			target = d.Name
			break
		}
	}
	if target == "" {
		t.Skip("durmuş distro yok; çalışan bir distroya dokunulmuyor")
	}

	if err := Start(ctx, target); err != nil {
		t.Fatalf("Start(%s): %v", target, err)
	}

	if got := stateOf(ctx, t, target); got != StateRunning {
		t.Errorf("Start sonrası durum = %q, Running bekleniyordu", got)
	}

	if err := Terminate(ctx, target); err != nil {
		t.Fatalf("Terminate(%s): %v", target, err)
	}

	if got := stateOf(ctx, t, target); got != StateStopped {
		t.Errorf("Terminate sonrası durum = %q, Stopped bekleniyordu", got)
	}
}

func stateOf(ctx context.Context, t *testing.T, name string) State {
	t.Helper()

	ds, err := List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, d := range ds {
		if d.Name == name {
			return d.State
		}
	}
	t.Fatalf("%s listede yok", name)
	return StateUnknown
}

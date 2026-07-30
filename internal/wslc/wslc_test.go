package wslc

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestStreamLinesDeliversAllLines(t *testing.T) {
	r := strings.NewReader("bir\niki\nüç\n")
	ch := make(chan string, 10)

	go streamLines(context.Background(), r, ch)

	var got []string
	for line := range ch {
		got = append(got, line)
	}

	want := []string{"bir", "iki", "üç"}
	if len(got) != len(want) {
		t.Fatalf("%d satır alındı, %d bekleniyordu: %q", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("satır %d = %q, %q bekleniyordu", i, got[i], want[i])
		}
	}
}

// Kanal kapanmazsa arayüz tarafındaki okuyucu sonsuza kadar bekler.
func TestStreamLinesClosesChannel(t *testing.T) {
	ch := make(chan string, 4)
	go streamLines(context.Background(), strings.NewReader("tek\n"), ch)

	<-ch // "tek"
	select {
	case _, open := <-ch:
		if open {
			t.Error("akış bitti ama kanal kapanmadı")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("kanal kapanmadı: okuyucu sonsuza kadar beklerdi")
	}
}

// Kimse okumadığında ctx iptali goroutine'i serbest bırakmalı; aksi hâlde
// panel her kapandığında bir goroutine sızar.
func TestStreamLinesStopsOnCancel(t *testing.T) {
	// Tamponsuz kanal: ilk yazma, okuyan olmadığı için bloke olur.
	ch := make(chan string)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		streamLines(ctx, strings.NewReader("a\nb\nc\n"), ch)
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("ctx iptal edildi ama streamLines çıkmadı (goroutine sızıntısı)")
	}
}

func TestStreamLinesHandlesLongLine(t *testing.T) {
	long := strings.Repeat("x", 200*1024) // varsayılan 64 KiB sınırının üstünde
	ch := make(chan string, 4)

	go streamLines(context.Background(), strings.NewReader(long+"\n"), ch)

	got, ok := <-ch
	if !ok {
		t.Fatal("uzun satır düşürüldü")
	}
	if len(got) != len(long) {
		t.Errorf("satır uzunluğu = %d, %d bekleniyordu", len(got), len(long))
	}
}

// JSON şeması belgelenmediği için Text, beklenmeyen tiplerde kaydı düşürmeden
// devam etmeli.
func TestTextAcceptsVariousShapes(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"string", `{"Names":"web"}`, "web"},
		{"dizi", `{"Names":["web","api"]}`, "web, api"},
		{"sayı", `{"Names":42}`, "42"},
		{"null", `{"Names":null}`, ""},
		{"nesne", `{"Names":{"a":1}}`, ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got Container
			if err := json.Unmarshal([]byte(c.in), &got); err != nil {
				t.Fatalf("ayrıştırma hata verdi: %v", err)
			}
			if got.Names.String() != c.want {
				t.Errorf("Names = %q, %q bekleniyordu", got.Names, c.want)
			}
		})
	}
}

// Alan adları büyük/küçük harf farkıyla gelirse de tutmalı.
func TestContainerFieldsAreCaseInsensitive(t *testing.T) {
	var c Container
	if err := json.Unmarshal([]byte(`{"id":"abc","image":"nginx","state":"running"}`), &c); err != nil {
		t.Fatalf("ayrıştırma hata verdi: %v", err)
	}
	if c.ID.String() != "abc" || c.Image.String() != "nginx" {
		t.Errorf("alanlar eşleşmedi: %+v", c)
	}
	if !c.IsRunning() {
		t.Error("IsRunning false döndü")
	}
}

func TestContainerNameFallsBackToID(t *testing.T) {
	c := Container{ID: "abc123"}
	if c.Name() != "abc123" {
		t.Errorf("Name() = %q; ad yokken kimliğe düşmeliydi", c.Name())
	}

	c = Container{ID: "abc123", Names: "/web"}
	if c.Name() != "web" {
		t.Errorf("Name() = %q; baştaki eğik çizgi atılmalıydı", c.Name())
	}
}

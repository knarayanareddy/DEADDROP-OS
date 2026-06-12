package panic

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPanicTriggerWipes(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "private.pem")
	if err := os.WriteFile(p, []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	d := &Daemon{Paths: []string{p}}
	start := time.Now()
	d.Wipe(TriggerSignal)
	if time.Since(start) > time.Second {
		t.Fatal("wipe too slow")
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatalf("file still exists: %v", err)
	}
}

func TestInactivityTimerResetsOnActivity(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "private.pem")
	if err := os.WriteFile(p, []byte("secret"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	activity := make(chan struct{}, 1)
	done := make(chan struct{})
	d := &Daemon{Paths: []string{p}, Inactivity: 80 * time.Millisecond}
	go func() { d.RunWithActivity(ctx, make(chan Trigger), activity); close(done) }()
	time.Sleep(50 * time.Millisecond)
	activity <- struct{}{}
	time.Sleep(50 * time.Millisecond)
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("file wiped before reset timeout: %v", err)
	}
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("daemon did not fire after reset timeout")
	}
	if _, err := os.Stat(p); !os.IsNotExist(err) {
		t.Fatalf("file still exists after inactivity: %v", err)
	}
}

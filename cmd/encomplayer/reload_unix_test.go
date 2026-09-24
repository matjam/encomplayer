//go:build unix

package main

import (
	"errors"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestSignalReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "encomplayer.pid")
	remove, err := writePID(path)
	if err != nil {
		t.Fatal(err)
	}
	defer remove()

	got := make(chan os.Signal, 1)
	signal.Notify(got, syscall.SIGUSR1)
	defer signal.Stop(got)

	pid, err := signalReload(path)
	if err != nil || pid != os.Getpid() {
		t.Fatalf("signalReload = %d, %v", pid, err)
	}
	select {
	case <-got:
	case <-time.After(2 * time.Second):
		t.Fatal("SIGUSR1 never arrived")
	}
}

func TestSignalReloadStalePID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "encomplayer.pid")
	// PIDs this large are beyond every Unix default maximum.
	if err := os.WriteFile(path, []byte("99999999\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := signalReload(path); !errors.Is(err, errNotRunning) {
		t.Fatalf("stale pid error = %v, want errNotRunning", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Error("stale pid file was not cleaned up")
	}
}

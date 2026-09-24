//go:build unix

package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// signalReload sends SIGUSR1 to the player recorded at path.
func signalReload(path string) (int, error) {
	pid, err := readPID(path)
	if err != nil {
		return 0, err
	}
	err = syscall.Kill(pid, syscall.SIGUSR1)
	switch {
	case errors.Is(err, syscall.ESRCH):
		// The player exited without cleaning up, e.g. after a crash.
		_ = os.Remove(path)
		return 0, fmt.Errorf("%w (stale pid %d)", errNotRunning, pid)
	case err != nil:
		return 0, fmt.Errorf("signal pid %d: %w", pid, err)
	}
	return pid, nil
}

// notifyReload calls reload on every SIGUSR1, so an edited config or theme
// applies with `pkill -USR1 encomplayer`. It returns a function that stops
// listening.
func notifyReload(reload func()) func() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGUSR1)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-ch:
				reload()
			case <-done:
				return
			}
		}
	}()
	return func() {
		signal.Stop(ch)
		close(done)
	}
}

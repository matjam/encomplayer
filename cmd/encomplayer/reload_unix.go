//go:build unix

package main

import (
	"os"
	"os/signal"
	"syscall"
)

// notifyReload calls reload on every SIGUSR1, so an edited config or theme
// applies with `pkill -USR1 encomplayer` as well as `encomplayer reload`.
// It returns a function that stops listening.
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

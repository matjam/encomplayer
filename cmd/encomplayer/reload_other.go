//go:build !unix

package main

import "errors"

// notifyReload is a no-op where there is no SIGUSR1; :reload covers it.
func notifyReload(func()) func() { return func() {} }

// signalReload is unavailable without SIGUSR1.
func signalReload(string) (int, error) {
	return 0, errors.New("--reload needs SIGUSR1, which this platform lacks; use :reload inside the player")
}

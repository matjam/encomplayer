//go:build !unix

package main

// notifyReload is a no-op where there is no SIGUSR1; the reload command
// reaches the player over the control socket instead.
func notifyReload(func()) func() { return func() {} }

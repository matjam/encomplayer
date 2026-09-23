//go:build !unix

package main

// notifyReload is a no-op where there is no SIGUSR1; :reload covers it.
func notifyReload(func()) func() { return func() {} }

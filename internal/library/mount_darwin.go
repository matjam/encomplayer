package library

import (
	"slices"

	"golang.org/x/sys/unix"
)

var networkFS = []string{"smbfs", "nfs", "afpfs", "webdav", "cifs", "macfuse", "osxfuse"}

// IsNetworkMount reports whether path lives on a network filesystem.
func IsNetworkMount(path string) bool {
	var st unix.Statfs_t
	if unix.Statfs(path, &st) != nil {
		return false
	}
	return slices.Contains(networkFS, unix.ByteSliceToString(st.Fstypename[:]))
}

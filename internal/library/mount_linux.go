package library

import (
	"slices"

	"golang.org/x/sys/unix"
)

// Filesystem magic numbers from statfs(2) for network and FUSE mounts.
var networkFS = []int64{
	0x6969,     // NFS
	0x517B,     // SMB
	0xFF534D42, // CIFS
	0xFE534D42, // SMB2
	0x65735546, // FUSE (sshfs, rclone)
	0x5346414F, // AFS
}

// IsNetworkMount reports whether path lives on a network filesystem.
func IsNetworkMount(path string) bool {
	var st unix.Statfs_t
	if unix.Statfs(path, &st) != nil {
		return false
	}
	return slices.Contains(networkFS, int64(st.Type))
}

////go:build windows
// +build windows

package utils

import "os"

// platformLockFile is a no-op on Windows to enable cross-compilation.
// TODO: Implement Windows file locking (e.g., via LockFileEx) if/when needed.
func platformLockFile(fd *os.File, write bool, nonblocking bool) error {
	return nil
}

// platformUnlockFile is a no-op on Windows to enable cross-compilation.
func platformUnlockFile(fd *os.File) error {
	return nil
}
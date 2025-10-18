/*
Copyright 2025

Unix-specific file locking using syscall.Flock.

This file provides low-level helpers for file locking on Unix-like systems.
Other code in the package can call these helpers to implement cross-platform
locking without importing syscall directly.

Build constraints:
- Included on: darwin, linux, freebsd, netbsd, openbsd, dragonfly, solaris
*/
////go:build darwin || linux || freebsd || netbsd || openbsd || dragonfly || solaris
// +build darwin linux freebsd netbsd openbsd dragonfly solaris

package utils

import (
	"os"
	"syscall"
)

// platformLockFile applies an advisory lock on the given file descriptor.
//
// Parameters:
// - fd: open file handle to lock
// - write: when true, request an exclusive lock (LOCK_EX); otherwise shared (LOCK_SH)
// - nonblocking: when true, add LOCK_NB to avoid blocking if the lock isn't immediately available
//
// Returns:
// - error: nil on success, or an error from syscall.Flock on failure
func platformLockFile(fd *os.File, write bool, nonblocking bool) error {
	if fd == nil {
		return nil
	}

	lockType := syscall.LOCK_SH // shared lock by default
	if write {
		lockType = syscall.LOCK_EX
	}
	if nonblocking {
		lockType |= syscall.LOCK_NB
	}

	return syscall.Flock(int(fd.Fd()), lockType)
}

// platformUnlockFile releases any lock held on the given file descriptor.
//
// Returns:
// - error: nil on success, or an error from syscall.Flock on failure
func platformUnlockFile(fd *os.File) error {
	if fd == nil {
		return nil
	}
	return syscall.Flock(int(fd.Fd()), syscall.LOCK_UN)
}
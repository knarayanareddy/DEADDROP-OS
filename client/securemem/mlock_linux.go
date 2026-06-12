//go:build linux

package securemem

import "syscall"

// LockProcessMemory asks the kernel not to swap current/future process pages.
// It is best-effort: unprivileged live sessions may lack RLIMIT_MEMLOCK.
func LockProcessMemory() error {
	return syscall.Mlockall(syscall.MCL_CURRENT | syscall.MCL_FUTURE)
}
